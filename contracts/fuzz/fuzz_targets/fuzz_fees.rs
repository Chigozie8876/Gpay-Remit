#![no_main]

use arbitrary::Arbitrary;
use gpay_remit_contracts::payment_escrow::{PaymentEscrowContract, PaymentEscrowContractClient};
use libfuzzer_sys::fuzz_target;
use soroban_sdk::{testutils::Address as _, Address, Env};

#[derive(Debug, Arbitrary)]
struct Input {
    amount: i128,
    platform: i128,
    forex: i128,
    compliance: i128,
    min_fee: i128,
    max_fee: i128,
}

fuzz_target!(|i: Input| {
    let env = Env::default();
    env.mock_all_auths();
    let admin = Address::generate(&env);
    let id = env.register_contract(None, PaymentEscrowContract);
    let client = PaymentEscrowContractClient::new(&env, &id);
    client.init_escrow(&admin);

    // Setters may reject out-of-range values; that is fine, they must not panic the host.
    let _ = client.try_set_platform_fee(&admin, &i.platform);
    let _ = client.try_set_forex_fee(&admin, &i.forex);
    let _ = client.try_set_compliance_fee(&admin, &i.compliance);
    let _ = client.try_set_fee_limits(&admin, &i.min_fee, &i.max_fee);

    if let Ok(Ok(fb)) = client.try_get_fee_breakdown(&i.amount) {
        assert!(fb.total_fee >= 0);
        assert!(fb.total_fee <= i.amount, "fee must never exceed the amount");
    }
});
