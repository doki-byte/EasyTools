/**
 * 免责声明与网络安全法协议：记录同意时间，超过有效期需重新确认。
 */
export const AGREEMENT_AT_KEY = "cybersecurity_agreement_at";
export const AGREEMENT_LEGACY_KEY = "cybersecurity_agreement";
export const AGREEMENT_VALID_MS = 7 * 24 * 60 * 60 * 1000;

export function isCybersecurityAgreementValid() {
  const raw = localStorage.getItem(AGREEMENT_AT_KEY);
  if (!raw) {
    localStorage.removeItem(AGREEMENT_LEGACY_KEY);
    return false;
  }
  const ts = Number(raw);
  if (!Number.isFinite(ts)) return false;
  return Date.now() - ts < AGREEMENT_VALID_MS;
}

export function saveCybersecurityAgreementTimestamp() {
  localStorage.setItem(AGREEMENT_AT_KEY, String(Date.now()));
  localStorage.removeItem(AGREEMENT_LEGACY_KEY);
}
