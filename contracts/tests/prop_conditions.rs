#![cfg(all(test, feature = "prop"))]
//! Property tests for escrow release-condition evaluation
//! (`PaymentEscrowContract::verify_conditions`).
//!
//! ## Shrinking
//!
//! Inputs are generated from local types that implement [`Arbitrary`]
//! (`ArbOperator`, `ArbConditionType`, `ArbCondition`, `EscrowScenario`).
//! Their strategies are built so a failing case shrinks to the smallest
//! input that still fails:
//!
//! * enums shrink toward their simplest variant (`And`, `Timestamp`);
//! * `required` shrinks toward `false`;
//! * proof / threshold values shrink toward `0`, the boundary
//!   `verify_conditions` cares about, not toward the range's lower bound;
//! * condition lists shrink by dropping conditions down to one;
//! * counters (approvals) shrink toward `0`.
//!
//! `shrinks_to_minimal_counterexample` pins this behaviour with a fixed seed.
//!
//! ## Reproducing a CI failure
//!
//! `proptest!` applies `PROPTEST_CASES` and `PROPTEST_RNG_SEED` from the
//! environment on top of the config below. CI sets a fixed seed per run and
//! prints it, so a failing run can be replayed exactly:
//!
//! ```text
//! PROPTEST_RNG_SEED=<seed from the CI log> PROPTEST_CASES=256 \
//!   cargo test --features prop --test prop_conditions
//! ```
//!
//! Failing inputs are also persisted to `proptest-regressions/prop_conditions.txt`,
//! which is replayed first on every later run.

use gpay_remit_contracts::payment_escrow::{
    Asset, ConditionOperator, ConditionType, PaymentEscrowContract, PaymentEscrowContractClient,
};
use proptest::prelude::*;
use proptest::strategy::BoxedStrategy;
use soroban_sdk::{
    testutils::{Address as _, Ledger},
    Address, Env, String,
};
use std::fmt;

// --- Helpers ---

fn setup_escrow_env(env: &Env) -> (PaymentEscrowContractClient<'_>, Address, Asset) {
    let contract_id = env.register_contract(None, PaymentEscrowContract);
    let client = PaymentEscrowContractClient::new(env, &contract_id);
    let admin = Address::generate(env);
    client.init_escrow(&admin);

    let asset = Asset {
        code: String::from_str(env, "USDC"),
        issuer: Address::generate(env),
    };
    client.add_supported_asset(&admin, &asset);

    (client, admin, asset)
}

fn create_base_escrow(
    env: &Env,
    client: &PaymentEscrowContractClient<'_>,
    asset: &Asset,
    expiration: u64,
) -> u64 {
    client.create_escrow(
        &Address::generate(env),
        &Address::generate(env),
        &1000,
        asset,
        &expiration,
        &String::from_str(env, ""),
    )
}

// --- Primitive strategies ---

/// An `i128` in `[min, max]` (with `min <= 0 <= max`) that shrinks toward 0.
///
/// A plain `min..=max` range shrinks toward `min`; for oracle prices and
/// proofs the interesting boundary is 0 (non-positive proofs never satisfy
/// an `OraclePrice` condition), so try the non-negative half first.
fn zero_centred_i128(min: i128, max: i128) -> BoxedStrategy<i128> {
    debug_assert!(min <= 0 && 0 <= max);
    if min == 0 {
        return (0..=max).boxed();
    }
    prop_oneof![0..=max, min..0].boxed()
}

/// A `u64` timestamp biased toward the edges (0, `u64::MAX`), shrinking to 0.
fn boundary_u64() -> BoxedStrategy<u64> {
    prop_oneof![Just(0u64), Just(u64::MAX), any::<u64>()].boxed()
}

// --- Arbitrary types ---

/// Condition operator. Shrinks to `And`.
///
/// Property: `verify_conditions` must agree with the reference model for
/// both operators (see `test_mixed_composability`, `test_empty_conditions`).
#[derive(Clone, Copy, PartialEq, Eq)]
struct ArbOperator(ConditionOperator);

impl fmt::Debug for ArbOperator {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{:?}", self.0)
    }
}

impl Arbitrary for ArbOperator {
    type Parameters = ();
    type Strategy = BoxedStrategy<Self>;

    fn arbitrary_with(_: ()) -> Self::Strategy {
        prop_oneof![Just(ConditionOperator::And), Just(ConditionOperator::Or)]
            .prop_map(ArbOperator)
            .boxed()
    }
}

/// Condition type, ordered so shrinking moves toward the condition that
/// depends on the fewest inputs (`Timestamp` depends only on ledger time).
///
/// Property: each type is evaluated from exactly one input (ledger time,
/// approvals, proof or KYC status), so a minimal counterexample names the
/// single input that matters.
#[derive(Clone, Copy, PartialEq, Eq)]
struct ArbConditionType(ConditionType);

impl fmt::Debug for ArbConditionType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{:?}", self.0)
    }
}

impl Arbitrary for ArbConditionType {
    type Parameters = ();
    type Strategy = BoxedStrategy<Self>;

    fn arbitrary_with(_: ()) -> Self::Strategy {
        prop_oneof![
            Just(ConditionType::Timestamp),
            Just(ConditionType::KYCVerified),
            Just(ConditionType::Approval),
            Just(ConditionType::MultiSignature),
            Just(ConditionType::OraclePrice),
        ]
        .prop_map(ArbConditionType)
        .boxed()
    }
}

/// One release condition. Shrinks toward an optional `Timestamp` with
/// threshold 0.
///
/// Property: `threshold_value` only matters for `OraclePrice`
/// (`proof >= threshold` when `proof > 0`); `required` decides whether a
/// failing condition fails an `And` escrow.
#[derive(Clone, PartialEq, Eq)]
struct ArbCondition {
    condition_type: ConditionType,
    required: bool,
    threshold_value: i128,
}

impl fmt::Debug for ArbCondition {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            f,
            "{:?}{}(t={})",
            self.condition_type,
            if self.required { "!" } else { "?" },
            self.threshold_value
        )
    }
}

impl Arbitrary for ArbCondition {
    type Parameters = ();
    type Strategy = BoxedStrategy<Self>;

    fn arbitrary_with(_: ()) -> Self::Strategy {
        (
            any::<ArbConditionType>(),
            any::<bool>(),
            zero_centred_i128(-100, 2000),
        )
            .prop_map(
                |(ArbConditionType(condition_type), required, threshold_value)| ArbCondition {
                    condition_type,
                    required,
                    threshold_value,
                },
            )
            .boxed()
    }
}

/// A complete escrow scenario: conditions, operator and every input the
/// conditions read. Used by `test_mixed_composability`.
///
/// Property: `verify_conditions(proof).all_passed` equals
/// [`EscrowScenario::expected_all_passed`] for every scenario.
///
/// Shrinks to one condition, `And`, proof 0, ledger time 0, no KYC and no
/// approvals — so a reported counterexample contains only the fields needed
/// to reproduce the failure.
#[derive(Clone, PartialEq, Eq)]
struct EscrowScenario {
    conditions: Vec<ArbCondition>,
    operator: ConditionOperator,
    proof: i128,
    ledger_time: u64,
    kyc_compliant: bool,
    approvals: u32,
    min_approvals: u32,
}

impl fmt::Debug for EscrowScenario {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            f,
            "{:?} {:?} proof={} ledger={} kyc={} approvals={}/{}",
            self.operator,
            self.conditions,
            self.proof,
            self.ledger_time,
            self.kyc_compliant,
            self.approvals,
            self.min_approvals
        )
    }
}

impl Arbitrary for EscrowScenario {
    type Parameters = ();
    type Strategy = BoxedStrategy<Self>;

    fn arbitrary_with(_: ()) -> Self::Strategy {
        (
            prop::collection::vec(any::<ArbCondition>(), 1..10),
            any::<ArbOperator>(),
            zero_centred_i128(-100, 2000),
            any::<u64>(),
            any::<bool>(),
            0u32..10,
            0u32..10,
        )
            .prop_map(
                |(
                    conditions,
                    ArbOperator(operator),
                    proof,
                    ledger_time,
                    kyc_compliant,
                    approvals,
                    min_approvals,
                )| {
                    EscrowScenario {
                        conditions,
                        operator,
                        proof,
                        ledger_time,
                        kyc_compliant,
                        approvals,
                        min_approvals,
                    }
                },
            )
            .boxed()
    }
}

impl EscrowScenario {
    /// Escrow expiration used by the scenario: 100 seconds before ledger
    /// time, so `Timestamp` conditions pass unless ledger time is < 100.
    fn expiration(&self) -> u64 {
        self.ledger_time.saturating_sub(100)
    }

    /// Reference model of `verify_conditions`.
    fn expected_all_passed(&self) -> bool {
        let expiration = self.expiration();
        let mut passed = 0;
        let mut required = 0;
        let mut failed_required = false;

        for c in &self.conditions {
            if c.required {
                required += 1;
            }
            let is_passed = match c.condition_type {
                ConditionType::Timestamp => self.ledger_time >= expiration,
                ConditionType::Approval | ConditionType::MultiSignature => {
                    self.approvals >= self.min_approvals
                }
                ConditionType::OraclePrice => self.proof > 0 && self.proof >= c.threshold_value,
                ConditionType::KYCVerified => self.kyc_compliant,
            };
            if is_passed {
                passed += 1;
            } else if c.required {
                failed_required = true;
            }
        }

        match self.operator {
            ConditionOperator::And => !failed_required && (required == 0 || passed >= required),
            ConditionOperator::Or => passed > 0,
        }
    }

    /// Run the scenario against the contract and return `all_passed`.
    fn run(&self) -> bool {
        let env = Env::default();
        env.mock_all_auths();
        env.ledger().with_mut(|li| li.timestamp = self.ledger_time);

        let (client, admin, asset) = setup_escrow_env(&env);
        let escrow_id = create_base_escrow(&env, &client, &asset, self.expiration());

        if self.kyc_compliant {
            client.admin_override_kyc(&admin, &escrow_id);
        }
        client.set_min_approvals(&escrow_id, &admin, &self.min_approvals);
        for _ in 0..self.approvals {
            client.add_approval(&escrow_id, &admin);
        }
        client.set_condition_operator(&escrow_id, &admin, &self.operator);
        for c in &self.conditions {
            client.add_condition(
                &escrow_id,
                &admin,
                &c.condition_type,
                &c.required,
                &c.threshold_value,
            );
        }

        client.verify_conditions(&escrow_id, &self.proof).all_passed
    }
}

// --- Tests ---

proptest! {
    // 1000 cases locally. `proptest!` applies the PROPTEST_* environment
    // variables on top of this config, so CI's PROPTEST_CASES and
    // PROPTEST_RNG_SEED take precedence.
    #![proptest_config(ProptestConfig::with_cases(1000))]

    /// Property: verifying twice with the same proof gives the same result
    /// (verification has no side effects on the outcome).
    #[test]
    fn test_verify_idempotency(
        expiration in boundary_u64(),
        ledger_time in boundary_u64(),
        proof_data in zero_centred_i128(i128::MIN, i128::MAX),
        operator in any::<ArbOperator>(),
    ) {
        let env = Env::default();
        env.mock_all_auths();
        env.ledger().with_mut(|li| li.timestamp = ledger_time);

        let (client, admin, asset) = setup_escrow_env(&env);
        let escrow_id = create_base_escrow(&env, &client, &asset, expiration);

        client.set_condition_operator(&escrow_id, &admin, &operator.0);

        let result1 = client.verify_conditions(&escrow_id, &proof_data);
        let result2 = client.verify_conditions(&escrow_id, &proof_data);

        prop_assert_eq!(result1.all_passed, result2.all_passed);
        prop_assert_eq!(result1.failed_conditions.len(), result2.failed_conditions.len());
    }

    /// Property: a timestamp condition that passes at time T still passes at
    /// any later time T + delta (time only moves an escrow toward release).
    #[test]
    fn test_monotonicity_timestamp(
        initial_time in 0u64..1_000_000_000u64,
        delta in 0u64..1_000_000_000u64,
        expiration in 0u64..2_000_000_000u64,
    ) {
        let env = Env::default();
        env.mock_all_auths();

        let (client, admin, asset) = setup_escrow_env(&env);
        let escrow_id = create_base_escrow(&env, &client, &asset, expiration);

        client.set_min_approvals(&escrow_id, &admin, &0);
        client.add_condition(&escrow_id, &admin, &ConditionType::Timestamp, &true, &0);

        env.ledger().with_mut(|li| li.timestamp = initial_time);
        let res_t = client.verify_conditions(&escrow_id, &0).all_passed;

        env.ledger().with_mut(|li| li.timestamp = initial_time + delta);
        let res_t_plus = client.verify_conditions(&escrow_id, &0).all_passed;

        if res_t {
            prop_assert!(res_t_plus, "passed at T={} but failed at T+{}", initial_time, delta);
        }
    }

    /// Property: with `And`, required oracle-price conditions pass exactly
    /// when the proof meets every threshold.
    #[test]
    fn test_composability_and_operator(
        thresholds in prop::collection::vec(0i128..1000i128, 1..5),
        proof in 0i128..2000i128,
    ) {
        let env = Env::default();
        env.mock_all_auths();

        let (client, admin, asset) = setup_escrow_env(&env);
        let escrow_id = create_base_escrow(&env, &client, &asset, 0);

        client.set_condition_operator(&escrow_id, &admin, &ConditionOperator::And);
        client.set_min_approvals(&escrow_id, &admin, &0);

        let mut expected_pass = true;
        for threshold in &thresholds {
            client.add_condition(&escrow_id, &admin, &ConditionType::OraclePrice, &true, threshold);
            if proof < *threshold {
                expected_pass = false;
            }
        }

        let result = client.verify_conditions(&escrow_id, &proof);
        prop_assert_eq!(result.all_passed, expected_pass, "proof={}, thresholds={:?}", proof, thresholds);
    }

    /// Property: with no conditions, `And` passes (empty conjunction) and
    /// `Or` fails (empty disjunction).
    #[test]
    fn test_empty_conditions(operator in any::<ArbOperator>()) {
        let env = Env::default();
        env.mock_all_auths();
        let (client, admin, asset) = setup_escrow_env(&env);
        let escrow_id = create_base_escrow(&env, &client, &asset, 0);

        client.set_condition_operator(&escrow_id, &admin, &operator.0);

        let result = client.verify_conditions(&escrow_id, &0);
        match operator.0 {
            ConditionOperator::And => prop_assert!(result.all_passed),
            ConditionOperator::Or => prop_assert!(!result.all_passed),
        }
    }

    /// Property: a required timestamp condition passes iff
    /// `ledger_time >= expiration`, including at 0 and `u64::MAX`.
    #[test]
    fn test_timestamp_extremes(
        expiration in boundary_u64(),
        ledger_time in boundary_u64(),
    ) {
        let env = Env::default();
        env.mock_all_auths();
        env.ledger().with_mut(|li| li.timestamp = ledger_time);

        let (client, admin, asset) = setup_escrow_env(&env);
        let escrow_id = create_base_escrow(&env, &client, &asset, expiration);

        client.set_min_approvals(&escrow_id, &admin, &0);
        client.add_condition(&escrow_id, &admin, &ConditionType::Timestamp, &true, &0);

        let result = client.verify_conditions(&escrow_id, &0);
        prop_assert_eq!(
            result.all_passed,
            ledger_time >= expiration,
            "ledger_time={}, expiration={}", ledger_time, expiration
        );
    }

    /// Property: for any mix of condition types, operator and inputs, the
    /// contract agrees with the reference model `expected_all_passed`.
    #[test]
    fn test_mixed_composability(scenario in any::<EscrowScenario>()) {
        prop_assert_eq!(scenario.run(), scenario.expected_all_passed(), "scenario: {:?}", scenario);
    }
}

/// The custom strategies must shrink a failure to its minimal form.
///
/// Uses a deliberately false property ("no condition has a threshold of 500
/// or more") with a fixed seed, so the result is deterministic, and checks
/// that proptest reports a single condition at exactly the boundary with
/// every unrelated field at its simplest value.
#[test]
fn shrinks_to_minimal_counterexample() {
    use proptest::test_runner::{Config, RngAlgorithm, TestError, TestRng, TestRunner};

    let mut runner = TestRunner::new_with_rng(
        Config {
            cases: 256,
            failure_persistence: None,
            ..Config::default()
        },
        TestRng::deterministic_rng(RngAlgorithm::ChaCha),
    );

    let result = runner.run(&any::<EscrowScenario>(), |s| {
        prop_assert!(s.conditions.iter().all(|c| c.threshold_value < 500));
        Ok(())
    });

    match result {
        Err(TestError::Fail(_, minimal)) => {
            assert_eq!(
                minimal.conditions.len(),
                1,
                "shrunk to one condition: {:?}",
                minimal
            );
            let c = &minimal.conditions[0];
            assert_eq!(
                c.threshold_value, 500,
                "threshold shrunk to the boundary: {:?}",
                minimal
            );
            assert_eq!(c.condition_type, ConditionType::Timestamp, "{:?}", minimal);
            assert!(!c.required, "{:?}", minimal);
            assert_eq!(minimal.operator, ConditionOperator::And, "{:?}", minimal);
            assert_eq!(minimal.proof, 0, "{:?}", minimal);
            assert_eq!(minimal.ledger_time, 0, "{:?}", minimal);
            assert!(!minimal.kyc_compliant, "{:?}", minimal);
            assert_eq!(
                (minimal.approvals, minimal.min_approvals),
                (0, 0),
                "{:?}",
                minimal
            );
        }
        other => panic!(
            "expected the property to fail and shrink, got {:?}",
            other.map(|_| ())
        ),
    }
}

/// Without custom shrinking, the old `-100..2000` proof range shrank toward
/// -100. The zero-centred strategy must reach 0 instead.
#[test]
fn zero_centred_strategy_shrinks_to_zero() {
    use proptest::test_runner::{Config, RngAlgorithm, TestError, TestRng, TestRunner};

    let mut runner = TestRunner::new_with_rng(
        Config {
            cases: 256,
            failure_persistence: None,
            ..Config::default()
        },
        TestRng::deterministic_rng(RngAlgorithm::ChaCha),
    );
    let result = runner.run(&zero_centred_i128(-100, 2000), |v| {
        prop_assert!(v == i128::MIN); // always fails
        Ok(())
    });
    match result {
        Err(TestError::Fail(_, v)) => assert_eq!(v, 0),
        other => panic!("expected failure, got {:?}", other.map(|_| ())),
    }
}
