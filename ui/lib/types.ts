export type GenesisData = {
  name: string;
  encryptionSharedParams: EGParams;
  trusteesRequired: number; // int
  registrar: RegistrarData;
  candidates: CandidateData[];
  trustees: TrusteeData[];
  timing: TimingData;
};

// base64url encoded bigint max 512 characters. enough for 3072bit numbers
// ours are max 2048bit for V1
type EncodedInts<T extends string> = Record<T, string>;

export type EGParams = EncodedInts<"g" | "p" | "q">;
export type PublicKey = EncodedInts<"y">;
export type PoK = EncodedInts<"m" | "c" | "r">;
export type Sig = EncodedInts<"c" | "r">;

// The dates could be in the future, so we want to be sure that
// we understand exactly the intended date, irrespective of changes
// to timezones. That is why this is not a direct UTC timestamp.
// We can convert it to a timestamp, and if in the past we will be
// more or less accurate (some times happen twice, but under those
// circumstances we agree to choose the "earliest" time. If its in
// the future, we can make a reasonable guess as to when it will happen
// based on our current timezone data.
// we will need a proper library to deal with this, so we have `luxon`.
// Actually the timezone is done separately for force it to be consistent
// across all the times.

export type TimeSpec = string; // "YYYY-mm-dd\Thh:MM:ss" RFC3339 without TZ

export type TimeWindow = {
  opens: TimeSpec; // DateTime
  closes: TimeSpec; // DateTime
};

export type TimingData = {
  timeZone: string; // the timezone for all this data.
  parameterConfirmation: TimeWindow;
  voterRegistration: TimeWindow;
  voteCasting: TimeWindow;
  tallyDecryption: TimeWindow;
};

type RegistrarData = {
  registrarId: string;
  name: string;
  verificationKey: PublicKey | null;
  encryptionKey: PublicKey | null;
  encryptionProof: PoK | null;
  eligibilityDataURL: string | null;
  eligibilityDataHash: string | null;
  registrationURL: string | null;
};

type TrusteeData = {
  trusteeId: string;
  name: string;
  verificationKey: PublicKey | null;
  publicExponents: string[] | null;
  signature: Sig | null;
};

type CandidateData = {
  candidateId: string;
  name: string;
};

type GlobalMeta = {
  version: string;
  commit: string;
  buildDate: string;
};

declare global {
  interface Window {
    META: GlobalMeta;
  }
}
