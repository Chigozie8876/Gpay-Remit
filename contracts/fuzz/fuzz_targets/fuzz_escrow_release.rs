#![no_main]

use arbitrary::Arbitrary;
use gpay_remit_contracts::payment_escrow::{Asset, PaymentEscrowContract, PaymentEscrowContractClient};
use libfuzzer_sys::fuzz_target;
use soroban_sdk::{
    testutils::{Address as _, Ledger},
    token, Address, Env, String,
};

#[derive(Debug, Arbitrary)]
struct Input {
    amount: u32,
    fee_bps: u16,
}

fuzz_target!(|i: Input| {
    let amount = i.amount as i128 + 1;
    let fee_bps = (i.fee_bps % 10_001) as i128;

    let env = Env::default();
    env.mock_all_auths();
    env.ledger().with_mut(|li| li.timestamp = 1_000);
    let admin = Address::generate(&env);
    let sender = Address::generate(&env);
    let recipient = Address::generate(&env);
    let treasury = Address::generate(&env);

    let id = env.register_contract(None, PaymentEscrowContract);
    let client = PaymentEscrowContractClient::new(&env, &id);
    client.init_escrow(&admin);

    let sac = env.register_stellar_asset_contract_v2(admin.clone());
    let token = token::Client::new(&env, &sac.address());
    token::StellarAssetClient::new(&env, &sac.address()).mint(&sender, &amount);
    let asset = Asset { code: String::from_str(&env, "USDC"), issuer: admin.clone() };
    client.add_supported_asset(&admin, &asset);
    client.set_platform_fee(&admin, &fee_bps);
    client.set_fee_wallet(&admin, &treasury);

    let eid = client.create_escrow(&sender, &recipient, &amount, &asset, &10_000, &String::from_str(&env, "f"));
    client.deposit(&eid, &sender, &amount, &token.address);
    if client.try_release_escrow(&eid, &recipient, &token.address).is_ok() {
        // Conservation of funds: nothing is created or lost on release.
        assert_eq!(token.balance(&recipient) + token.balance(&treasury), amount);
        assert_eq!(token.balance(&id), 0);
    }
});
