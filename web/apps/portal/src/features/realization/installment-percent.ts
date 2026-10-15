const DISPLAY_DECIMALS = 6;
const scale = 10n ** BigInt(DISPLAY_DECIMALS);

function fail(): never {
  throw new Error(
    `Укажите первоначальный взнос больше 0% и меньше 100% (до ${DISPLAY_DECIMALS} знаков после запятой).`,
  );
}

/** Converts a user-entered percent to minor units, rounding positive halves up. */
export function percentToMinor(percent: string, priceMinor: string): string {
  if (!/^\d{1,38}$/.test(priceMinor) || BigInt(priceMinor) <= 0n) fail();
  const match = /^(\d{1,3})(?:\.(\d{1,6}))?$/.exec(percent);
  if (!match) fail();
  const whole = BigInt(match[1]!);
  const fraction = BigInt((match[2] ?? '').padEnd(DISPLAY_DECIMALS, '0') || '0');
  const percentScaled = whole * scale + fraction;
  if (percentScaled <= 0n || percentScaled >= 100n * scale) fail();
  const divisor = 100n * scale;
  const amount = (BigInt(priceMinor) * percentScaled + divisor / 2n) / divisor;
  if (amount <= 0n || amount >= BigInt(priceMinor)) fail();
  return amount.toString();
}

/** Shows a money-origin amount as a percent rounded to six decimal places. */
export function minorToPercent(amountMinor: string, priceMinor: string): string {
  if (!/^\d{1,38}$/.test(amountMinor) || !/^\d{1,38}$/.test(priceMinor) || BigInt(priceMinor) <= 0n) return '';
  const scaled = (BigInt(amountMinor) * 100n * scale + BigInt(priceMinor) / 2n) / BigInt(priceMinor);
  const whole = scaled / scale;
  const fraction = (scaled % scale).toString().padStart(DISPLAY_DECIMALS, '0').replace(/0+$/, '');
  return fraction ? `${whole}.${fraction}` : whole.toString();
}

export const percentDisplayPrecision = DISPLAY_DECIMALS;
