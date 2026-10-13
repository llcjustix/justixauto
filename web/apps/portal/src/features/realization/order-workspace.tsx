import type { Order } from './data';
import { vehicleColorsLabel } from './vehicle-colors';

/** UUIDs are unreadable in headings and links; the first block is enough to tell orders and shipments apart. */
export const shortId = (id: string): string => id.slice(0, 8);

export function orderLineLabel(line: Order['terms']['lines'][number], name: (id: string) => string): string {
  return `${name(line.modelId)} · ${vehicleColorsLabel(line)}`;
}

export function nextOrderAction(o: Order): { label: string; message: string; needsMe: boolean } {
  const pending = o.addenda.find((a) => a.status === 'proposed');
  if (pending && pending.proposedBy !== o.party && o.allowedActions.includes('accept-addendum'))
    return { label: 'Рассмотреть изменение', message: 'Ваш ответ: рассмотрите предложенное изменение условий.', needsMe: true };
  if (o.allowedActions.includes('confirm'))
    return { label: 'Подтвердить заказ', message: 'Ваш ответ: подтвердите или отклоните заказ покупателя.', needsMe: true };
  if (o.party === 'buyer' && o.receiptBatches?.some((b) => Number(b.unidentifiedCount) > 0))
    return { label: 'Ввести VIN', message: 'Ваш следующий шаг: введите недостающие VIN в поступившие партии.', needsMe: true };
  if (o.party === 'buyer' && o.allowedActions.includes('set-warehouse') && !o.receivingWarehouseId)
    return { label: 'Выбрать склад получения', message: 'Ваш следующий шаг: выберите склад получения для автоматического поступления.', needsMe: true };
  if (o.allowedActions.includes('ship-quantity') || o.allowedActions.includes('ship') || o.allowedActions.includes('allocate'))
    return { label: 'Продолжить отгрузку', message: 'Ваш следующий шаг: подготовьте и отгрузите автомобили.', needsMe: true };
  if (o.party === 'buyer' && o.shipments.some((s) => s.status !== 'received'))
    return { label: 'Открыть приёмку', message: 'Ваш следующий шаг: откройте прежнюю отгрузку для приёмки.', needsMe: true };
  if (pending) return { label: 'Открыть', message: `Ждём ${o.party === 'buyer' ? 'поставщика' : 'покупателя'}: решение по вашему изменению условий.`, needsMe: false };
  if (o.status === 'awaiting-supplier') return { label: 'Открыть', message: 'Ждём поставщика: подтверждение заказа.', needsMe: false };
  if (o.party === 'buyer' && ['accepted', 'fulfilling'].includes(o.status))
    return { label: 'Открыть', message: 'Ждём поставщика: следующая отгрузка. Оплаты доступны во вкладке «Оплаты».', needsMe: false };
  return { label: 'Открыть', message: ['completed', 'cancelled', 'rejected'].includes(o.status)
    ? 'Заказ завершён. Документы и история доступны во вкладках.' : 'Ожидается действие контрагента. Проверьте оплаты и историю.', needsMe: false };
}
