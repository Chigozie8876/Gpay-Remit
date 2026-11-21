import "@testing-library/jest-dom";

// Mock Ledger hardware wallet transport — WebUSB is not available in jsdom.
jest.mock("@ledgerhq/hw-transport-webusb", () => ({
  __esModule: true,
  default: { create: jest.fn().mockResolvedValue({ close: jest.fn() }) },
}));

jest.mock("@ledgerhq/hw-app-str", () => ({
  __esModule: true,
  default: jest.fn().mockImplementation(() => ({
    getPublicKey: jest.fn().mockResolvedValue({ publicKey: "GTEST..." }),
    signTransaction: jest.fn().mockResolvedValue({ signature: Buffer.from("sig") }),
  })),
}));

// Mock Freighter browser extension API — not available in jsdom.
jest.mock("@stellar/freighter-api", () => ({
  isConnected: jest.fn().mockResolvedValue(false),
  getPublicKey: jest.fn().mockResolvedValue("GTEST..."),
  signTransaction: jest.fn().mockResolvedValue("signedXDR"),
  requestAccess: jest.fn().mockResolvedValue(undefined),
}));

// Mock albedo-link — opens a popup window, not available in jsdom.
jest.mock("@albedo-link/intent", () => ({
  __esModule: true,
  default: {
    publicKey: jest.fn().mockResolvedValue({ pubkey: "GTEST..." }),
    tx: jest.fn().mockResolvedValue({ signed_envelope_xdr: "signedXDR" }),
  },
}));
