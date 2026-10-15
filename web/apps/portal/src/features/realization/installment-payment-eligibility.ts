import type { InstallmentPlan, RetailInvoice } from './data';

export function eligibleInstallmentInvoices(dealId: string, plan: InstallmentPlan, invoices: RetailInvoice[]) {
  if (plan.state === 'cancelled' || !plan.allowedActions?.includes('submit-installment-payment')) return [];
  const byId = new Map(invoices.map((invoice) => [invoice.id, invoice]));
  return plan.rows
    .map((row) => ({ row, invoice: byId.get(row.invoiceId) }))
    .filter((value): value is { row: InstallmentPlan['rows'][number]; invoice: RetailInvoice } => {
      const { row, invoice } = value;
      return (
        !!invoice &&
        invoice.dealId === dealId &&
        invoice.purpose === 'monthly-installment' &&
        invoice.status === 'issued' &&
        invoice.installmentNumber === row.number &&
        invoice.amount.amountMinor === row.amount.amountMinor &&
        row.allowedActions?.includes('submit-payment') &&
        invoice.allowedActions?.includes('submit-payment') &&
        [row.amount, row.available, invoice.amount, invoice.available, invoice.pending, invoice.outstanding].every(
          (amount) => amount.currency === plan.scheduledTotal.currency,
        ) &&
        BigInt(invoice.available.amountMinor) > 0n &&
        BigInt(row.available.amountMinor) > 0n
      );
    });
}
