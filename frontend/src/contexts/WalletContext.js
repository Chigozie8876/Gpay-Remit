import React, { createContext, useContext, useState, useMemo, useCallback } from 'react';
import { LedgerWalletService } from '../services/wallets/ledger.ts';
import { FreighterWalletService } from '../services/wallets/freighter.ts';
import { AlbedoWalletService } from '../services/wallets/albedo.ts';
import { XBullWalletService } from '../services/wallets/xbull.ts';

/** @typedef {'ledger'|'freighter'|'albedo'|'xbull'} WalletType */

const WalletContext = createContext();

export const useWallet = () => {
  const context = useContext(WalletContext);
  if (!context) {
    throw new Error('useWallet must be used within a WalletProvider');
  }
  return context;
};

export const WalletProvider = ({ children }) => {
  const [publicKey, setPublicKey]       = useState(null);
  const [walletType, setWalletType]     = useState(null);
  const [isConnecting, setIsConnecting] = useState(false);
  const [error, setError]               = useState(null);

  // Instantiate adapters once; they hold no open resources until connect() is called.
  const adapters = useMemo(() => ({
    ledger:    new LedgerWalletService(),
    freighter: new FreighterWalletService(),
    albedo:    new AlbedoWalletService(),
    xbull:     new XBullWalletService(),
  }), []);

  const getAdapter = useCallback((type) => {
    const adapter = adapters[type];
    if (!adapter) throw new Error(`Unknown wallet type: ${type}`);
    return adapter;
  }, [adapters]);

  /**
   * Connect to the given wallet type.
   * @param {WalletType} type
   */
  const connect = useCallback(async (type) => {
    setIsConnecting(true);
    setError(null);
    try {
      const adapter = getAdapter(type);
      const result  = await adapter.connect();
      setPublicKey(result.publicKey);
      setWalletType(type);
    } catch (err) {
      setError(err.message || `Failed to connect to ${type}`);
    } finally {
      setIsConnecting(false);
    }
  }, [getAdapter]);

  /** Disconnect whichever wallet is currently active. */
  const disconnect = useCallback(async () => {
    if (walletType) {
      try {
        await getAdapter(walletType).disconnect();
      } catch {
        // Ignore disconnect errors; always clear local state.
      }
    }
    setPublicKey(null);
    setWalletType(null);
    setError(null);
  }, [walletType, getAdapter]);

  /**
   * Sign a transaction XDR with the active wallet.
   * @param {string} txXdr  Base-64 XDR string of the transaction envelope.
   * @param {string} [networkPassphrase]
   * @returns {Promise<string>} Signed XDR.
   */
  const signTransaction = useCallback(async (txXdr, networkPassphrase) => {
    if (!walletType) throw new Error('No wallet connected');
    return getAdapter(walletType).signTransaction(txXdr, networkPassphrase);
  }, [walletType, getAdapter]);

  // Keep backwards-compatible connectLedger helper.
  const connectLedger = useCallback(() => connect('ledger'), [connect]);

  const value = {
    publicKey,
    walletType,
    isConnecting,
    error,
    connect,
    connectLedger,
    disconnect,
    signTransaction,
    isConnected: !!publicKey,
  };

  return (
    <WalletContext.Provider value={value}>
      {children}
    </WalletContext.Provider>
  );
};
