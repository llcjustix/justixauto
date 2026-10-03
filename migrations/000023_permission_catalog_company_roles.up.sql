-- User decisions 2026-09-26:
-- * The permission catalog lives in PostgreSQL: which permissions can be put
--   into roles, their scope and their Russian name. New permissions are added
--   here directly; code decides what each permission unlocks.
-- * Platform roles (platform permissions) are prepared in Admin; each company
--   creates its own private roles (company permissions) in its cabinet.

CREATE TABLE identity_permissions (
    key        text PRIMARY KEY CHECK (length(key) BETWEEN 3 AND 100),
    scope      text NOT NULL CHECK (scope IN ('platform', 'company')),
    name       text NOT NULL CHECK (length(name) BETWEEN 1 AND 200),
    assignable boolean NOT NULL DEFAULT true
);

INSERT INTO identity_permissions (key, scope, name, assignable) VALUES
    ('platform.companies.create',     'platform', 'Платформа: создание компаний', false),
    ('platform.companies.access',     'platform', 'Платформа: доступ и удаление компаний', false),
    ('platform.users.manage',         'platform', 'Платформа: сотрудники платформы', false),
    ('platform.memberships.manage',   'platform', 'Платформа: членство в компаниях', false),
    ('platform.roles.manage',         'platform', 'Платформа: роли и разрешения', false),
    ('platform.directory.read',       'platform', 'Платформа: справочник компаний', true),
    ('platform.audit.read',           'platform', 'Платформа: журнал действий', true),
    ('company.create',                'company',  'Компания: создание', true),
    ('company.edit',                  'company',  'Компания: изменение реквизитов', true),
    ('company.users.manage',          'company',  'Компания: сотрудники и роли', true),
    ('branches.create',               'company',  'Филиалы: создание', true),
    ('branches.edit',                 'company',  'Филиалы: изменение', true),
    ('inventory.read',                'company',  'Склад: просмотр', true),
    ('inventory.models.edit',         'company',  'Склад: модели автомобилей', true),
    ('inventory.warehouses.manage',   'company',  'Склад: управление складами', true),
    ('inventory.receipts.create',     'company',  'Склад: приёмка автомобилей', true),
    ('inventory.vehicles.move',       'company',  'Склад: перемещение автомобилей', true),
    ('commerce.read',                 'company',  'Закупки: просмотр', true),
    ('commerce.partnerships.manage',  'company',  'Закупки: партнёрства', true),
    ('commerce.offers.manage',        'company',  'Закупки: предложения', true),
    ('commerce.trade',                'company',  'Закупки: оптовые сделки', true),
    ('commerce.payments.accept',      'company',  'Закупки: подтверждение оплат', true),
    ('retail.read',                   'company',  'Продажи: просмотр', true),
    ('retail.crm.manage',             'company',  'Продажи: CRM (лиды, клиенты, задачи)', true),
    ('retail.listings.manage',        'company',  'Продажи: витрина', true),
    ('retail.deals.manage',           'company',  'Продажи: сделки', true),
    ('retail.payments.accept',        'company',  'Продажи: приём оплат', true),
    ('retail.deals.deliver',          'company',  'Продажи: выдача автомобиля', true),
    ('financing.read',                'company',  'Финансирование: просмотр', true),
    ('financing.programs.manage',     'company',  'Финансирование: программы', true),
    ('financing.applications.manage', 'company',  'Финансирование: подача заявок', true),
    ('financing.applications.agree',  'company',  'Финансирование: согласие клиента', true),
    ('financing.applications.review', 'company',  'Финансирование: рассмотрение заявок', true),
    ('financing.applications.decide', 'company',  'Финансирование: решение по заявкам', true),
    ('insurance.read',                'company',  'Страхование: просмотр', true),
    ('insurance.applications.manage', 'company',  'Страхование: подача заявок', true),
    ('insurance.applications.review', 'company',  'Страхование: рассмотрение заявок', true),
    ('insurance.applications.decide', 'company',  'Страхование: решение по заявкам', true),
    ('documents.read',                'company',  'Документы: просмотр', true),
    ('documents.upload',              'company',  'Документы: загрузка', true),
    ('documents.sensitive.download',  'company',  'Документы: скачивание персональных данных', true);

-- A company's own roles: private to it, company permissions only.
ALTER TABLE identity_roles
    ADD COLUMN company_id uuid REFERENCES identity_companies (id),
    ADD CONSTRAINT roles_company_scope_check CHECK (company_id IS NULL OR scope = 'company');

-- Role names are unique among platform roles and within each company.
DROP INDEX roles_name_key;
CREATE UNIQUE INDEX roles_name_key ON identity_roles (lower(name)) WHERE company_id IS NULL;
CREATE UNIQUE INDEX roles_company_name_key ON identity_roles (company_id, lower(name)) WHERE company_id IS NOT NULL;
