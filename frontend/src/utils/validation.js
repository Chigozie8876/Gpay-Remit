// #110 — client-side validation utilities.

const STELLAR_ADDRESS_RE = /^G[A-Z2-7]{55}$/;
const CURRENCY_CODES = new Set(["USD", "EUR", "GBP", "XLM", "NGN", "KES", "GHS"]);

/** #357 — true when `address` is a well-formed Stellar public key (G + 55 base32 chars). */
export function isValidStellarAddress(address) {
  return typeof address === "string" && STELLAR_ADDRESS_RE.test(address.trim());
}

export function validateStellarAddress(address) {
  if (!address) return "Stellar address is required.";
  if (!isValidStellarAddress(address))
    return "Invalid Stellar address (must start with G, be 56 characters, and use only A-Z and 2-7).";
  return null;
}

export function validateAmount(amount) {
  const num = parseFloat(amount);
  if (!amount && amount !== 0) return "Amount is required.";
  if (isNaN(num)) return "Amount must be a number.";
  if (num <= 0) return "Amount must be greater than zero.";
  if (num > 1_000_000) return "Amount exceeds the maximum of 1,000,000.";
  if (!/^\d+(\.\d{1,7})?$/.test(String(amount)))
    return "Amount can have at most 7 decimal places.";
  return null;
}

export function validateCurrency(code) {
  if (!code) return "Currency is required.";
  if (!CURRENCY_CODES.has(code.toUpperCase()))
    return `Unsupported currency code: ${code}.`;
  return null;
}

export function validateUserId(id) {
  if (!id && id !== 0) return "User ID is required.";
  if (!/^\d+$/.test(String(id))) return "User ID must be a positive integer.";
  return null;
}

/** Validate the remittance form. Returns a map of field → error string. */
export function validateRemittanceForm(data) {
  const errors = {};
  const senderId = validateUserId(data.sender_id);
  if (senderId) errors.sender_id = senderId;
  const recipientId = validateUserId(data.recipient_id);
  if (recipientId) errors.recipient_id = recipientId;
  const amount = validateAmount(data.amount);
  if (amount) errors.amount = amount;
  const currency = validateCurrency(data.currency);
  if (currency) errors.currency = currency;
  const targetCurrency = validateCurrency(data.target_currency);
  if (targetCurrency) errors.target_currency = targetCurrency;
  const recipientAccount = validateStellarAddress(data.recipient_account);
  if (recipientAccount) errors.recipient_account = recipientAccount;
  if (data.currency && data.target_currency && data.currency === data.target_currency)
    errors.target_currency = "Source and target currencies must differ.";
  if (data.notes && data.notes.length > 500)
    errors.notes = "Notes cannot exceed 500 characters.";
  return errors;
}
