import albedo from "@albedo-link/intent";

export class AlbedoWalletService {
  private publicKey: string | null = null;

  async connect(): Promise<{ publicKey: string }> {
    const result = await albedo.publicKey({ require_existing: false });
    if (!result?.pubkey) {
      throw new Error("Albedo did not return a public key");
    }
    this.publicKey = result.pubkey;
    return { publicKey: result.pubkey };
  }

  async disconnect(): Promise<void> {
    this.publicKey = null;
  }

  async signTransaction(txXdr: string, network?: "public" | "testnet"): Promise<string> {
    const result = await albedo.tx({
      xdr: txXdr,
      network: network ?? "public",
      submit: false,
    });
    if (!result?.signed_envelope_xdr) {
      throw new Error("Albedo did not return a signed transaction");
    }
    return result.signed_envelope_xdr;
  }
}
