#![cfg(test)]

use gpay_remit_contracts::payment_escrow::{
    Asset, EscrowStatus, PaymentEscrowContract, PaymentEscrowContractClient,
};
use proptest::prelude::*;
use soroban_sdk::{
    testutils::{Address as _, Ledger},
    token, Address, Env, String,
};

proptest! {
    /// For any amount, a funded escrow must hold exactly what was deposited and
    /// release must never create or destroy tokens (recipient + fee wallet == amount).
    #[test]
    fn fuzz_deposit_and_release_conserves_funds(
        amount in 1i128..1_000_000_000_000_000i128,
        fee_bps in 0i128..=10_000i128,
    ) {
        let env = Env::default();
        env.mock_all_auths();
        env.ledger().with_mut(|li| li.timestamp = 1_000);

        let admin = Address::generate(&env);
        let sender = Address::generate(&env);
        let recipient = Address::generate(&env);
        let treasury = Address::generate(&env);

        let contract_id = env.register_contract(None, PaymentEscrowContract);
        let client = PaymentEscrowContractClient::new(&env, &contract_id);
        client.init_escrow(&admin);

        let sac = env.register_stellar_asset_contract_v2(admin.clone());
        let token = token::Client::new(&env, &sac.address());
        token::StellarAssetClient::new(&env, &sac.address()).mint(&sender, &amount);

        let asset = Asset {
            code: String::from_str(&env, "USDC"),
            issuer: admin.clone(),
        };
        client.add_supported_asset(&admin, &asset);
        client.set_platform_fee(&admin, &fee_bps);
        client.set_fee_wallet(&admin, &treasury);

        let escrow_id = client.create_escrow(
            &sender,
            &recipient,
            &amount,
            &asset,
            &10_000,
            &String::from_str(&env, "fuzz"),
        );
        client.deposit(&escrow_id, &sender, &amount, &token.address);

        let escrow = client.get_escrow(&escrow_id).unwrap();
        prop_assert_eq!(escrow.deposited_amount, amount);
        prop_assert_eq!(escrow.status, EscrowStatus::Funded);
        prop_assert_eq!(token.balance(&contract_id), amount);

        // A release either succeeds (conserving funds) or fails without moving any.
        match client.try_release_escrow(&escrow_id, &recipient, &token.address) {
            Ok(_) => {
                prop_assert_eq!(
                    token.balance(&recipient) + token.balance(&treasury),
                    amount
                );
                prop_assert_eq!(token.balance(&contract_id), 0);
                prop_assert_eq!(token.balance(&treasury), amount * fee_bps / 10_000);
            }
            Err(_) => {
                prop_assert_eq!(token.balance(&contract_id), amount);
            }
        }
    }
}
