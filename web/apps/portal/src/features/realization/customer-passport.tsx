import { ActionButton, fileUrl } from '@justixauto/kit';
import type { FieldSpec } from '@justixauto/kit';
import type { Customer } from './data';

const customerFields = (customer?: Customer): FieldSpec[] => [
  { name: 'displayName', label: 'Имя', type: 'text', required: true, initial: customer?.displayName ?? '' },
  { name: 'phone', label: 'Телефон', type: 'text', initial: customer?.phone ?? '' },
  {
    name: 'passportSeries',
    label: 'Серия документа',
    type: 'text',
    group: 'Документ',
    initial: customer?.passportSeries ?? '',
  },
  {
    name: 'passportNumber',
    label: 'Номер документа',
    type: 'text',
    group: 'Документ',
    initial: customer?.passportNumber ?? '',
  },
  {
    name: 'passportIssuer',
    label: 'Кем выдан',
    type: 'text',
    group: 'Документ',
    full: true,
    initial: customer?.passportIssuer ?? '',
  },
  {
    name: 'passportIssuedOn',
    label: 'Дата выдачи',
    type: 'date',
    group: 'Документ',
    initial: customer?.passportIssuedOn ?? '',
  },
  {
    name: 'passportExpiresOn',
    label: 'Действителен до',
    type: 'date',
    group: 'Документ',
    initial: customer?.passportExpiresOn ?? '',
  },
  ...personalFields(customer),
  { name: 'passportCopy', label: 'Копия документа', type: 'file', purpose: 'passport-copy', group: 'Копия документа' },
];

const personalFields = (customer?: Customer): FieldSpec[] => [
  {
    name: 'birthDate',
    label: 'Дата рождения',
    type: 'date',
    group: 'Личные данные',
    initial: customer?.birthDate ?? '',
  },
  {
    name: 'personalId',
    label: 'Персональный ID',
    type: 'text',
    group: 'Личные данные',
    initial: customer?.personalId ?? '',
  },
  {
    name: 'registeredAddress',
    label: 'Адрес регистрации',
    type: 'textarea',
    group: 'Личные данные',
    initial: customer?.registeredAddress ?? '',
  },
];

function profile(values: Record<string, unknown>, customer?: Customer) {
  const { passportCopy, ...fields } = values;
  const fileID = typeof passportCopy === 'string' ? passportCopy : '';
  return {
    ...fields,
    passportCopyFileIds: fileID
      ? [...(customer?.passportCopyFileIds ?? []), fileID]
      : (customer?.passportCopyFileIds ?? []),
  };
}

export function CustomerPassportAction({
  customer,
  label,
  onSubmit,
}: {
  customer?: Customer;
  label: string;
  onSubmit: (profile: Record<string, unknown>) => Promise<unknown>;
}) {
  const copies = customer?.passportCopyFileIds ?? [];
  return (
    <ActionButton
      small
      label={label}
      fields={customerFields(customer)}
      refresh={[['customers']]}
      intro={
        <>
          <p>Поля документа необязательны. Копия хранится приватно и открывается только после проверки доступа.</p>
          {copies.map((id, index) => (
            <p key={id}>
              <a href={fileUrl(id)} target="_blank" rel="noreferrer">
                Копия документа {index + 1}
              </a>
            </p>
          ))}
        </>
      }
      onSubmit={(values) => onSubmit(profile(values, customer))}
    />
  );
}
