import { ActionButton, ApiError, post, useRefresh, useSession } from '@justixauto/kit';
import type { Vehicle, Warehouse } from './data';

const movePermission = 'inventory.vehicles.move';

export function VehicleMoveAction({ vehicle, warehouses }: { vehicle: Vehicle; warehouses: Warehouse[] }) {
  const session = useSession();
  const refresh = useRefresh();
  const sourceID = vehicle.placement?.warehouseId;
  if (!sourceID || !session.can(movePermission)) return null;
  const targets = warehouses.filter((warehouse) => warehouse.id !== sourceID);
  if (!targets.length) return null;
  const refreshKeys = [
    ['vehicle', vehicle.id],
    ['vehicles'],
    ['warehouses'],
    ['stock', sourceID],
    ...targets.map((warehouse) => ['stock', warehouse.id]),
  ];
  return (
    <ActionButton
      label="Переместить"
      refresh={refreshKeys}
      fields={[
        {
          name: 'toWarehouseId',
          label: 'На склад',
          type: 'select',
          required: true,
          options: targets.map(
            (warehouse) => [warehouse.id, `${warehouse.name} (свободно ${warehouse.free})`] as [string, string],
          ),
        },
        { name: 'occurredAt', label: 'Когда', type: 'datetime', required: true },
      ]}
      onSubmit={async (values) => {
        try {
          return await post(`/inventory/vehicle-units/${vehicle.id}/warehouse-moves`, {
            fromWarehouseId: sourceID,
            toWarehouseId: values.toWarehouseId,
            occurredAt: values.occurredAt,
            evidenceBindingIds: [],
          });
        } catch (error) {
          if (error instanceof ApiError && error.code === 'placement_changed') {
            try {
              await refresh(...refreshKeys);
            } catch {
              /* preserve the move error for the dialog */
            }
          }
          throw error;
        }
      }}
    />
  );
}
