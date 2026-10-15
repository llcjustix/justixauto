import { call } from '@justixauto/kit';
import type { ConsignmentReturn } from './data';

const PAGE_SIZE = 100;
type ReturnPage = { items: ConsignmentReturn[]; nextCursor?: string | null };

/** The list helper discards nextCursor, so order history reads the list envelope itself. */
export async function loadConsignmentReturnHistory(orderId: string): Promise<ConsignmentReturn[]> {
  const rows: ConsignmentReturn[] = [];
  const seen = new Set<string>();
  const cursors = new Set<number>();
  let offset = 0;
  for (;;) {
    if (cursors.has(offset)) throw new Error('История возвратов содержит повторяющуюся страницу.');
    cursors.add(offset);
    const page = (await call<unknown>(
      'GET',
      `/commerce/consignments/return-requests?orderId=${encodeURIComponent(orderId)}&limit=${PAGE_SIZE}&offset=${offset}`,
    )) as unknown as ReturnPage;
    if (!Array.isArray(page.items)) throw new Error('Не удалось прочитать историю возвратов.');
    for (const row of page.items) {
      if (!row.id || seen.has(row.id)) throw new Error('История возвратов содержит повторяющуюся запись.');
      seen.add(row.id);
      rows.push(row);
    }
    if (page.nextCursor == null) return rows;
    if (!/^\d+$/.test(page.nextCursor)) throw new Error('История возвратов содержит неверный указатель страницы.');
    const next = Number(page.nextCursor);
    if (page.items.length !== PAGE_SIZE || !Number.isSafeInteger(next) || next !== offset + PAGE_SIZE)
      throw new Error('История возвратов содержит пропуск страницы.');
    offset = next;
  }
}
