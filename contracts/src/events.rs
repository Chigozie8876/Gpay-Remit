use soroban_sdk::{contracttype, symbol_short, Address, Env, String, Symbol};

#[derive(Clone, Debug, PartialEq, Eq)]
#[contracttype]
pub struct AssetRef {
    pub code: String,
    pub issuer: Address,
}

#[derive(Clone, Debug, PartialEq, Eq)]
#[contracttype]
pub enum EventData {
    EscrowCreated(u64, Address, Address, AssetRef, i128),
    EscrowDeposited(u64, i128, i128),
    EscrowApproved(u64),
    EscrowReleased(u64, i128),
    EscrowRefunded(u64, i128),
    EscrowExtended(u64, u64),
    InvoiceCreated(u64, u64, Address, Address, AssetRef, i128),
    InvoicePaid(u64, u64, i128),
    InvoiceUpdated(u64, i128, i128),
    InvoiceCancelled(u64),
    InvoiceOverdue(u64),
    AdminAction(Symbol),
    AddressAction(Symbol, Address),
    PairAction(Symbol, Address, Address),
}

#[cfg_attr(not(test), derive(Clone, Debug, PartialEq, Eq))]
#[cfg_attr(not(test), contracttype)]
#[cfg_attr(test, derive(Clone, Debug, PartialEq, Eq))]
pub struct GpayEvent {
    /// Ledger close time (seconds since epoch) of the ledger that emitted the event.
    pub timestamp: u64,
    /// Ledger sequence number of the ledger that emitted the event. Together with
    /// the event's position within that ledger this gives indexers a total order.
    pub ledger: u32,
    pub actor: Address,
    pub amount: i128,
    pub status: Symbol,
    pub data: EventData,
}

pub fn emit(
    env: &Env,
    component: Symbol,
    action: Symbol,
    id: u64,
    actor: &Address,
    amount: i128,
    status: Symbol,
    #[allow(unused_variables)] data: EventData,
) {
    #[cfg(not(test))]
    {
        env.events().publish(
            (symbol_short!("gpayremit"), component, action, id),
            GpayEvent {
                timestamp: env.ledger().timestamp(),
                ledger: env.ledger().sequence(),
                actor: actor.clone(),
                amount,
                status,
                data,
            },
        );
    }
    #[cfg(test)]
    {
        // In test mode, publish a simpler event structure
        env.events().publish(
            (symbol_short!("gpayremit"), component, action, id),
            (
                env.ledger().timestamp(),
                env.ledger().sequence(),
                actor.clone(),
                amount,
                status,
            ),
        );
    }
}
