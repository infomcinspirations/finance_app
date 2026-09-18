// Money helpers. Cents are the only representation that crosses the wire; these
// functions exist so no other module is tempted to do arithmetic on a float.

/**
 * Parses user input such as "12.50", "-8", or "1,234.56" into signed cents.
 * Returns null when the input is not a well-formed amount with at most two
 * decimal places, so the caller can show a validation message.
 */
export function parseAmountToCents(input: string): number | null {
  const s = input.trim().replace(/[,\s]/g, "");
  if (s === "" || s === "-" || s === "." || s === "-.") return null;
  if (!/^-?\d*(\.\d{0,2})?$/.test(s)) return null;

  const negative = s.startsWith("-");
  const body = negative ? s.slice(1) : s;
  const [whole = "", frac = ""] = body.split(".");

  // Pad so "5.5" reads as 50 cents rather than 5.
  const cents = Number(whole || "0") * 100 + Number((frac + "00").slice(0, 2));
  if (!Number.isSafeInteger(cents)) return null;

  return negative ? -cents : cents;
}

/** Formats signed cents for display, e.g. -1250 with "USD" -> "-$12.50". */
export function formatCents(cents: number, currency = "USD"): string {
  try {
    return new Intl.NumberFormat(undefined, {
      style: "currency",
      currency,
    }).format(cents / 100);
  } catch {
    // An unrecognised currency code should not blank out the number.
    return `${(cents / 100).toFixed(2)} ${currency}`;
  }
}

/** Formats cents without a currency symbol, for input fields. */
export function centsToInput(cents: number): string {
  const sign = cents < 0 ? "-" : "";
  const abs = Math.abs(cents);
  return `${sign}${Math.floor(abs / 100)}.${String(abs % 100).padStart(2, "0")}`;
}

/** Today as YYYY-MM-DD in the viewer's own timezone, for date inputs. */
export function todayISODate(): string {
  const now = new Date();
  const local = new Date(now.getTime() - now.getTimezoneOffset() * 60_000);
  return local.toISOString().slice(0, 10);
}

/** Renders an API timestamp as a short local date. */
export function formatDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}
