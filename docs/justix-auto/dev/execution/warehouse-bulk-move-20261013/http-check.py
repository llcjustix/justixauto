#!/usr/bin/env python3
"""Focused disposable HTTP fixture/check driver; no browser automation.

Run seed-legacy against the copied schema45 API, then upgrade the owned runtime,
then check. --case permits a narrowly selected, not-yet-run final scenario.
Never imports or calls internal/e2e, never accepts arbitrary API/DB targets.
"""
import argparse
from concurrent.futures import ThreadPoolExecutor
from copy import deepcopy
from datetime import datetime, timezone
import hashlib
import http.cookiejar
import importlib.util
import json
import os
from pathlib import Path
import re
import secrets
import selectors
import subprocess
import sys
import threading
import time
import urllib.error
import urllib.request
import uuid

sys.dont_write_bytecode = True
_spec = importlib.util.spec_from_file_location('warehouse_runtime', Path(__file__).with_name('runtime-control.py'))
rt = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(rt)
STATE = None
EVENTS = []
SNAPSHOT_TABLES = None
EVENT_LOCK = threading.Lock()


def now():
    return datetime.now(timezone.utc).isoformat(timespec='seconds').replace('+00:00', 'Z')


def check(ok, message):
    rt.require(ok, message)


def record(name, **data):
    with EVENT_LOCK:
        EVENTS.append({'at': now(), 'name': name, **data})
        if STATE:
            rt.private_write(Path(STATE['directory']) / 'http-evidence.json', EVENTS)


class Client:
    def __init__(self, account=None):
        self.jar = http.cookiejar.CookieJar()
        self.http = urllib.request.build_opener(urllib.request.ProxyHandler({}), urllib.request.HTTPCookieProcessor(self.jar))
        self.csrf = ''
        self.account = account
        if account:
            login = self.do('POST', '/identity/session/login',
                            {'login': account['login'], 'password': account['password']}, expected=200)
            if login['data']['user'].get('passwordChangeRequired'):
                password = secrets.token_urlsafe(24)
                self.do('POST', '/identity/session/password', {'currentPassword': account['password'],
                        'newPassword': password, 'newPasswordConfirmation': password}, expected=200)
                account['password'] = password
                rt.save(STATE)
            self.do('GET', '/identity/session', expected=200)
            if account.get('companyId'):
                session = self.do('GET', '/identity/session', expected=200)
                self.do('PUT', '/identity/session/context', {'companyId': account['companyId']},
                        revision=session['revision'], expected=200)

    def do(self, method, path, body=None, revision=None, expected=None, code=None, timeout=5):
        check(path.startswith('/') and not path.startswith('//') and '://' not in path, 'Unexpected HTTP path')
        headers = {'Content-Type': 'application/json', 'Origin': rt.API}
        if self.csrf:
            headers['X-CSRF-Token'] = self.csrf
        if revision is not None:
            headers['If-Match'] = '"' + str(revision) + '"'
        req = urllib.request.Request(rt.API + '/api/v1' + path,
                data=None if body is None else json.dumps(body).encode(), headers=headers, method=method)
        started = time.monotonic()
        try:
            response = self.http.open(req, timeout=timeout)
        except urllib.error.HTTPError as response_error:
            response = response_error
        with response:
            status = response.status
            raw = response.read()
            token = response.headers.get('X-CSRF-Token')
            if token:
                self.csrf = token
        payload = json.loads(raw) if raw else {}
        error_code = payload.get('error', {}).get('code')
        # No credentials, Cookie/CSRF headers, login bodies or full session bodies in evidence.
        record('http', method=method, path=path, status=status, error=error_code,
               elapsedMs=round((time.monotonic() - started) * 1000))
        if expected is not None:
            check(status == expected, f'{method} {path}: expected {expected}, got {status}/{error_code}')
        if code is not None:
            check(error_code == code, f'{method} {path}: expected error {code}, got {error_code}')
        payload['_status'] = status
        return payload

    def get(self, path):
        return self.do('GET', path, expected=200)


def account(name):
    return Client(STATE['accounts'][name])


def warehouse(client, name, capacity):
    return client.do('POST', '/inventory/warehouses', {'name': name,
        'country': {'key': 'UZ', 'label': 'Uzbekistan'}, 'city': 'Tashkent',
        'address': 'Synthetic warehouse fixture', 'capacity': str(capacity)}, expected=201)['data']['id']


def wh(client, wid):
    return client.get('/inventory/warehouses/' + wid)['data']


def batch(client, bid):
    return client.get('/inventory/receipt-batches/' + bid)['data']


def receipt(client, wid, quantity=None, vins=None):
    stock = {'mode': 'identified', 'vins': vins} if vins else {'mode': 'unidentified', 'quantity': str(quantity)}
    return client.do('POST', f'/inventory/warehouses/{wid}/receipt-batches', {
        'modelId': STATE['fixture']['model'], 'modelSpecificationVersion': '1',
        'exteriorColor': 'white', 'interiorColor': 'black', 'receivedAt': now(), 'stock': stock},
        revision=wh(client, wid)['revision'], expected=201)['data']


def company(admin, name, user_type):
    credentials = {'login': 'warehouse-' + name.lower(), 'password': secrets.token_urlsafe(24)}
    STATE['accounts'][name] = credentials
    rt.save(STATE)  # preserve generated password before any network effect
    created = admin.do('POST', '/identity/admin/seller-companies', {'userType': user_type,
        'company': {'name': 'Warehouse E2E ' + name, 'country': {'key': 'UZ', 'label': 'Uzbekistan'},
                    'registration': 'WH-E2E-' + name, 'email': name.lower() + '@warehouse-e2e.test'},
        'firstAdmin': {'displayName': name, 'login': credentials['login'],
                       'password': credentials['password'], 'passwordConfirmation': credentials['password']}}, expected=201)['data']
    cid = created['company']['id']
    credentials.update(companyId=cid, userId=created['admin']['id'])
    rt.save(STATE)
    activated = admin.do('POST', f'/identity/admin/companies/{cid}/activate', {'reason': 'Disposable verification fixture'},
                         revision=created['company']['revision'], expected=200)
    owner = Client(credentials)
    submitted = owner.do('POST', f'/identity/companies/{cid}/submit-verification', {},
                         revision=activated['revision'], expected=200)
    admin.do('POST', f'/identity/admin/companies/{cid}/verify', {}, revision=submitted['revision'], expected=200)
    return Client(credentials)


def seed_legacy():
    check(STATE['stage'] == 'legacy_ready', 'Legacy seed is single-use and requires schema45 runtime')
    STATE['stage'] = 'legacy_seeding'
    rt.save(STATE)
    admin = account('admin')
    buyer = company(admin, 'Buyer', 'dealer')
    supplier = company(admin, 'Supplier', 'distributor')
    foreign = company(admin, 'Foreign', 'dealer')
    model = admin.do('POST', '/inventory/vehicle-models', {'specification': {
        'make': 'Chevrolet', 'model': 'Warehouse E2E', 'variant': 'Bulk', 'year': 2026,
        'bodyType': 'sedan', 'exteriorColors': ['white'], 'interiorColors': ['black'],
        'exteriorColor': 'white', 'interiorColor': 'black', 'powertrain': 'petrol 1.5', 'drivetrain': 'FWD'}}, expected=201)['data']['id']
    STATE['fixture'] = {'model': model, 'warehouses': {}, 'batches': {}, 'vehicles': []}
    f = STATE['fixture']
    for key, capacity in [('source', 100), ('destination', 100), ('third', 100), ('tiny', 1)]:
        f['warehouses'][key] = warehouse(buyer, 'E2E ' + key, capacity)
    f['warehouses']['foreign'] = warehouse(foreign, 'E2E foreign', 20)
    source = f['warehouses']['source']
    r = receipt(buyer, source, quantity=8)
    f['batches']['manualLegacy'] = r['batch']['id']
    f['legacyBatchBefore'] = r['batch']
    r = receipt(buyer, source, vins=['XTAAA11111A700003', 'XTAAA11111A700001', 'XTAAA11111A700004', 'XTAAA11111A700002'])
    f['batches']['identified'] = r['batch']['id']
    f['vehicles'] = [v['id'] for v in r['vehicles']]
    r = receipt(foreign, f['warehouses']['foreign'], vins=['XTAAA11111A800001'])
    f['foreignVehicle'] = r['vehicles'][0]['id']
    f['batches']['foreign'] = receipt(foreign, f['warehouses']['foreign'], quantity=3)['batch']['id']
    partnership = supplier.do('POST', '/commerce/partnerships',
                             {'counterpartyCompanyId': STATE['accounts']['Buyer']['companyId']}, expected=201)
    buyer.do('POST', '/commerce/partnerships/' + partnership['data']['id'] + '/accept', {},
             revision=partnership['revision'], expected=200)
    order = buyer.do('POST', '/commerce/orders', {'supplierCompanyId': STATE['accounts']['Supplier']['companyId'],
        'warehouseId': source, 'terms': {'route': 'local', 'deliveryTerms': 'Synthetic receipt fixture',
            'warrantyTerms': '', 'serviceTerms': '', 'paymentSchedule': [],
            'lines': [{'modelId': model, 'modelSpecificationVersion': '1', 'exteriorColor': 'white',
                       'interiorColor': 'black', 'quantity': '6', 'unitPrice': {'amountMinor': '100000', 'currency': 'USD'}}]}}, expected=201)
    oid = order['data']['id']
    accepted = supplier.do('POST', f'/commerce/orders/{oid}/supplier-confirmations', {}, revision=order['revision'], expected=200)
    line = accepted['data']['terms']['lines'][0]['lineId']
    supplier.do('POST', f'/commerce/orders/{oid}/shipments', {'route': 'local',
        'lines': [{'orderLineId': line, 'quantity': '6', 'vins': [], 'modelSpecificationVersion': '1',
                   'exteriorColor': 'white', 'interiorColor': 'black'}]}, revision=accepted['revision'], expected=201)
    order = buyer.get('/commerce/orders/' + oid)['data']
    check(len(order['receiptBatches']) == 1, 'Expected exactly one order-origin batch')
    f.update(orderId=oid, orderLineId=line, orderBefore=order)
    f['batches']['order'] = order['receiptBatches'][0]['receiptBatchId']
    cid = STATE['accounts']['Buyer']['companyId']
    role = buyer.do('POST', f'/identity/companies/{cid}/roles',
                    {'name': 'Warehouse E2E reader', 'permissionKeys': ['inventory.read']}, expected=201)
    reader = {'login': 'warehouse-reader', 'password': secrets.token_urlsafe(24), 'companyId': cid}
    STATE['accounts']['Reader'] = reader
    rt.save(STATE)
    user = buyer.do('POST', f'/identity/companies/{cid}/users', {'displayName': 'Warehouse E2E reader',
        'login': reader['login'], 'password': reader['password'], 'roleIds': [role['data']['id']]}, expected=201)
    reader['userId'] = user['data']['id']
    STATE['stage'] = 'legacy_seeded'
    rt.save(STATE)
    record('legacy_seeded', orderId=oid, sourceWarehouseId=source, originalBatchIds=f['batches'])
    print('legacy_seeded: manual+order-origin receipts, four VINs, foreign tenant, read-only principal')


def stock_snapshot():
    global SNAPSHOT_TABLES
    if SNAPSHOT_TABLES is None:
        SNAPSHOT_TABLES = rt.sql(STATE, "SELECT tablename FROM pg_tables WHERE schemaname='public' AND (tablename LIKE 'inventory_%' OR tablename LIKE 'commerce_%') ORDER BY tablename;").splitlines()
    pieces = []
    for table in SNAPSHOT_TABLES:
        check(re.fullmatch(r'(inventory|commerce)_[a-z_]+', table), 'Unexpected fixture table identifier')
        pieces.append(f"SELECT '{table}' AS name, COALESCE(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text),'[]'::jsonb) AS content FROM {table} t")
    check(bool(pieces), 'Disposable stock schema missing')
    data = json.loads(rt.sql(STATE, 'SELECT jsonb_object_agg(name,content) FROM (' + ' UNION ALL '.join(pieces) + ') snapshots;'))
    return {table: hashlib.sha256(json.dumps(rows, sort_keys=True).encode()).hexdigest() for table, rows in data.items()}


def invariants():
    result = rt.sql(STATE, '''SELECT count(*) FROM inventory_receipt_batches b LEFT JOIN
      (SELECT receipt_batch_id,sum(unidentified_count) n FROM inventory_receipt_locations GROUP BY receipt_batch_id) l
      ON l.receipt_batch_id=b.id WHERE COALESCE(l.n,0)<>b.unidentified_count
      OR b.confirmed_quantity<>b.identified_count+b.unidentified_count;''')
    check(result == '0', 'Global/local receipt invariant broken')
    result = rt.sql(STATE, '''SELECT count(*) FROM inventory_warehouses w WHERE
      (SELECT count(*) FROM inventory_placements p WHERE p.warehouse_id=w.id)
      + COALESCE((SELECT sum(l.unidentified_count) FROM inventory_receipt_locations l WHERE l.warehouse_id=w.id),0)>w.capacity;''')
    check(result == '0', 'Warehouse capacity exceeded')


def move_payload(client, source, destination, vehicles, quantities):
    return ({'toWarehouseId': destination, 'toWarehouseRevision': wh(client, destination)['revision'],
             'occurredAt': now(), 'vehicleIds': vehicles, 'unidentified': [
                 {'receiptBatchId': bid, 'quantity': str(qty), 'revision': batch(client, bid)['batch']['revision']}
                 for bid, qty in quantities]}, wh(client, source)['revision'])


def move(client, source, body, revision, status=200, code=None):
    return client.do('POST', f'/inventory/warehouses/{source}/stock-moves', body, revision=revision,
                     expected=status, code=code, timeout=3)


def locations(data):
    return {row['warehouseId']: int(row['unidentifiedCount']) for row in data['locations']}


def reject(name, client, source, body, revision, status, code=None):
    before = stock_snapshot()
    move(client, source, body, revision, status, code)
    check(stock_snapshot() == before, name + ': failed command changed stock/commerce/facts')
    record('rollback_pass', case=name, expected=status, code=code)


def mixed(client):
    f, w, b = STATE['fixture'], STATE['fixture']['warehouses'], STATE['fixture']['batches']
    b['postMigration'] = receipt(client, w['source'], quantity=6)['batch']['id']
    rt.save(STATE)
    source_before, dest_before = wh(client, w['source']), wh(client, w['destination'])
    roots = {bid: batch(client, bid) for bid in [b['manualLegacy'], b['order']]}
    vehicles = {vid: client.get('/inventory/vehicle-units/' + vid)['data']['vehicle'] for vid in f['vehicles'][:2]}
    before_commerce = {k: v for k, v in stock_snapshot().items() if k.startswith('commerce_')}
    body, revision = move_payload(client, w['source'], w['destination'], f['vehicles'][:2],
                                  [(b['manualLegacy'], 3), (b['order'], 2)])
    f['mixedRequest'] = {'source': w['source'], 'revision': revision, 'body': body}
    rt.save(STATE)
    result = move(client, w['source'], body, revision)['data']
    check(set(result['movedVehicleIds']) == set(f['vehicles'][:2]), 'Moved VIN list differs')
    check(int(wh(client, w['source'])['occupied']) == int(source_before['occupied']) - 7, 'Source occupancy delta')
    check(int(wh(client, w['destination'])['occupied']) == int(dest_before['occupied']) + 7, 'Destination occupancy delta')
    check(int(wh(client, w['source'])['revision']) == int(source_before['revision']) + 1, 'Source revision must change once')
    check(int(wh(client, w['destination'])['revision']) == int(dest_before['revision']) + 1, 'Destination revision must change once')
    for bid, quantity in [(b['manualLegacy'], 3), (b['order'], 2)]:
        after = batch(client, bid)
        for key, value in roots[bid]['batch'].items():
            if key != 'revision':
                check(after['batch'][key] == value, 'Move changed global receipt field ' + key)
        check(locations(after)[w['destination']] == quantity, 'Destination pending balance')
        check(locations(after)[w['source']] == int(roots[bid]['batch']['unidentifiedCount']) - quantity, 'Source pending balance')
        check(int(after['batch']['revision']) == int(roots[bid]['batch']['revision']) + 1, 'Root receipt revision must change once')
        for wid in (w['source'], w['destination']):
            stock = client.get(f'/inventory/warehouses/{wid}/inventory')['data']
            listed = next(row for row in stock['unidentifiedBatches'] if row['id'] == bid)
            check(int(listed['warehouseUnidentifiedCount']) == locations(after)[wid], 'Warehouse-local DTO count differs')
    for vid, original in vehicles.items():
        detail = client.get('/inventory/vehicle-units/' + vid)['data']
        for key in ('id', 'vin', 'modelId', 'modelSpecificationVersion', 'exteriorColor', 'interiorColor', 'ownerCompanyId', 'custodianCompanyId', 'reserved'):
            check(detail['vehicle'][key] == original[key], 'Move changed vehicle field ' + key)
        check(detail['vehicle']['placement']['receiptBatchId'] == original['placement']['receiptBatchId'], 'VIN receipt provenance lost')
        check(detail['vehicle']['placement']['warehouseId'] == w['destination'], 'VIN destination wrong')
        check(any(row['type'] == 'vehicle.moved' for row in detail['history']), 'VIN movement history absent')
    after_commerce = {k: v for k, v in stock_snapshot().items() if k.startswith('commerce_')}
    check(before_commerce == after_commerce, 'Movement changed order/payment/commercial tables')
    quantity_facts = rt.sql(STATE, "SELECT count(*) FROM inventory_facts WHERE fact_type='receipt.quantity_moved';")
    check(quantity_facts == '2', 'Expected one movement fact per moved pending batch')
    reject('exact replay', client, w['source'], body, revision, 412, 'stale_revision')
    onwards, revision = move_payload(client, w['destination'], w['third'], [], [(b['manualLegacy'], 1)])
    move(client, w['destination'], onwards, revision)
    single = client.do('POST', f'/inventory/vehicle-units/{f["vehicles"][0]}/warehouse-moves',
        {'fromWarehouseId': w['destination'], 'toWarehouseId': w['third'], 'occurredAt': now()}, expected=200)['data']
    check(single['placement']['receiptBatchId'] == b['identified'], 'Legacy single-move receipt provenance lost')
    invariants()
    record('mixed_pass', units=7, replayRolledBack=True, quantityOnward=True, legacyMoveProvenance=True)


def rollback(client):
    f, w, b = STATE['fixture'], STATE['fixture']['warehouses'], STATE['fixture']['batches']
    source = w['source']
    base, revision = move_payload(client, source, w['destination'], [f['vehicles'][2]], [(b['manualLegacy'], 1)])
    cases = []
    def add(name, change, status, code=None, rev=revision):
        body = deepcopy(base)
        change(body)
        cases.append((name, client, body, rev, status, code))
    add('stale source', lambda body: None, 412, 'stale_revision', rev='1')
    add('stale destination', lambda body: body.update(toWarehouseRevision='1'), 412, 'stale_revision')
    add('stale batch', lambda body: body['unidentified'][0].update(revision='1'), 412, 'stale_revision')
    add('unknown VIN after valid', lambda body: body['vehicleIds'].append(str(uuid.uuid4())), 404)
    add('foreign VIN after valid', lambda body: body['vehicleIds'].append(f['foreignVehicle']), 404)
    add('wrong-source VIN after valid', lambda body: body['vehicleIds'].append(f['vehicles'][0]), 409, 'placement_changed')
    add('foreign batch after valid', lambda body: body['unidentified'].append({'receiptBatchId': b['foreign'], 'quantity': '1', 'revision': '1'}), 404)
    add('local quantity exceeded', lambda body: body['unidentified'][0].update(quantity='6'), 409, 'exceeds_unidentified')
    add('foreign destination', lambda body: body.update(toWarehouseId=w['foreign'], toWarehouseRevision='1'), 404)
    add('same destination', lambda body: body.update(toWarehouseId=source, toWarehouseRevision=revision), 422)
    add('empty selection', lambda body: body.update(vehicleIds=[], unidentified=[]), 422)
    add('duplicate VIN', lambda body: body['vehicleIds'].append(body['vehicleIds'][0]), 422)
    add('duplicate batch', lambda body: body['unidentified'].append(deepcopy(body['unidentified'][0])), 422)
    add('duplicate receipt UUID alias', lambda body: body['unidentified'].append(
        dict(body['unidentified'][0], receiptBatchId=uuid.UUID(body['unidentified'][0]['receiptBatchId']).urn)), 422)
    add('zero quantity', lambda body: body['unidentified'][0].update(quantity='0'), 422)
    # jsonx.Quantity decode errors are HTTP400; semantic validation is HTTP422.
    add('fractional quantity', lambda body: body['unidentified'][0].update(quantity='0.5'), 400)
    add('invalid occurrence', lambda body: body.update(occurredAt='not-a-date'), 400)
    add('future occurrence', lambda body: body.update(occurredAt='2099-01-01T00:00:00Z'), 422)
    add('missing precondition', lambda body: None, 428, 'if_match_required', rev=None)
    add('combined capacity', lambda body: body.update(toWarehouseId=w['tiny'], toWarehouseRevision=wh(client, w['tiny'])['revision']), 409, 'capacity_exceeded')
    cases.append(('missing permission', account('Reader'), deepcopy(base), revision, 403, 'permission_denied'))
    for name, actor, body, rev, status, code in cases:
        reject(name, actor, source, body, rev, status, code)
    invariants()


def identify(client):
    f, w, b = STATE['fixture'], STATE['fixture']['warehouses'], STATE['fixture']['batches']
    bid = b['order']
    initial = batch(client, bid)
    occupied = wh(client, w['destination'])['occupied']
    result = client.do('POST', f'/inventory/receipt-batches/{bid}/identifications',
        {'warehouseId': w['destination'], 'atomic': True, 'items': [{'vin': 'XTAAA11111A900001', 'modelId': f['model']}]}, expected=200)['data']
    check(result['batch']['id'] == bid and result['batch']['confirmedQuantity'] == initial['batch']['confirmedQuantity'], 'Identification changed original receipt')
    check(wh(client, w['destination'])['occupied'] == occupied, 'Identification changed occupancy')
    vid = result['vehicles'][0]['id']
    vehicle = client.get('/inventory/vehicle-units/' + vid)['data']['vehicle']
    check(vehicle['placement']['warehouseId'] == w['destination'] and vehicle['placement']['receiptBatchId'] == bid, 'Destination identification provenance')
    check(vehicle['exteriorColor'] == 'white' and vehicle['interiorColor'] == 'black', 'Identification changed exact colors')
    # Legacy request defaults to original source, even while other locations exist.
    result = client.do('POST', f'/inventory/receipt-batches/{bid}/identifications',
        {'atomic': True, 'items': [{'vin': 'XTAAA11111A900002', 'modelId': f['model']}]}, expected=200)['data']
    check(client.get('/inventory/vehicle-units/' + result['vehicles'][0]['id'])['data']['vehicle']['placement']['warehouseId'] == w['source'], 'Legacy identify default changed')
    remaining = locations(batch(client, bid))[w['source']]
    body, revision = move_payload(client, w['source'], w['destination'], [], [(bid, remaining)])
    move(client, w['source'], body, revision)
    order = client.get('/commerce/orders/' + f['orderId'])['data']
    p = next(x for x in order['lineProgress'] if x['orderLineId'] == f['orderLineId'])
    check(p['identified'] == '2' and p['unidentified'] == '4' and not p['receiptQuantityAdjusted'], 'Order identification counted incorrectly after original location drained')
    check(order['receiptBatches'][0]['receiptBatchId'] == bid, 'Order receipt ID changed')
    check(locations(batch(client, bid)).get(w['source'], 0) == 0 and locations(batch(client, bid))[w['destination']] == 4, 'Drained receipt location discovery incorrect')
    source_stock = client.get(f'/inventory/warehouses/{w["source"]}/inventory')['data']
    check(all(row['id'] != bid for row in source_stock['unidentifiedBatches']), 'Drained batch remains in source inventory')
    before = stock_snapshot()
    client.do('POST', f'/inventory/receipt-batches/{bid}/identifications', {'atomic': True,
        'items': [{'vin': 'XTAAA11111A900003', 'modelId': f['model']}]}, expected=409, code='exceeds_unidentified')
    check(before == stock_snapshot(), 'Failed legacy identify changed global/local state')
    invariants()
    record('identification_pass', orderId=f['orderId'], originalReceiptId=bid, destinationVIN=vid, originDrained=True)


def correction(client):
    f, w, b = STATE['fixture'], STATE['fixture']['warehouses'], STATE['fixture']['batches']
    bid = b['manualLegacy']
    original = batch(client, bid)
    target = w['third']
    local = locations(original)[target]
    before = stock_snapshot()
    client.do('POST', f'/inventory/receipt-batches/{bid}/quantity-corrections',
              {'warehouseId': target, 'quantity': str(int(original['batch']['confirmedQuantity']) - local - 1),
               'reason': 'Synthetic local lower-bound check'}, revision=original['batch']['revision'], expected=409, code='exceeds_unidentified')
    check(before == stock_snapshot(), 'Rejected correction fabricated stock elsewhere')
    result = client.do('POST', f'/inventory/receipt-batches/{bid}/quantity-corrections',
              {'warehouseId': target, 'quantity': str(int(original['batch']['confirmedQuantity']) - 1),
               'reason': 'Synthetic selected-location correction'}, revision=original['batch']['revision'], expected=200)
    after = batch(client, bid)
    check(int(after['batch']['confirmedQuantity']) == int(original['batch']['confirmedQuantity']) - 1, 'Global correction delta')
    for wid, quantity in locations(original).items():
        check(locations(after).get(wid, 0) == quantity - (1 if wid == target else 0), 'Correction touched wrong location')
    invariants()
    record('correction_pass', selectedWarehouse=target, originalReceiptId=result['data']['batch']['id'])


def held_lock(statement, callback):
    cid = rt.container(STATE)
    argv = ['docker', 'exec', '-i', cid, 'psql', '-X', '-qAt', '-v', 'ON_ERROR_STOP=1', '-U', 'postgres', '-d', rt.DB]
    proc = subprocess.Popen(argv, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, bufsize=1)
    proc.stdin.write("BEGIN; SET LOCAL statement_timeout='6s';\n\\o /dev/null\n" + statement +
                     ";\n\\o\n\\echo WAREHOUSE_LOCK_READY\nSELECT pg_sleep(4); ROLLBACK;\n")
    proc.stdin.close()
    selector = selectors.DefaultSelector()
    selector.register(proc.stdout, selectors.EVENT_READ)
    deadline = time.monotonic() + 3
    locked = False
    try:
        while time.monotonic() < deadline:
            if selector.select(0.1):
                line = proc.stdout.readline().strip()
                if line == 'WAREHOUSE_LOCK_READY':
                    locked = True
                    break
        check(locked, 'Controlled lock did not become ready')
        callback()
    finally:
        selector.close()
        proc.wait(timeout=8)  # SQL's bounded sleep/statement timeout releases its locks.
        check(proc.returncode == 0, 'Disposable lock holder failed')


def contention(client):
    f, w, b = STATE['fixture'], STATE['fixture']['warehouses'], STATE['fixture']['batches']
    body, revision = move_payload(client, w['source'], w['destination'], [f['vehicles'][2], f['vehicles'][3]], [(b['manualLegacy'], 1)])
    locks = [('target warehouse first', f"SELECT id FROM inventory_warehouses WHERE id='{uuid.UUID(w['destination'])}' FOR UPDATE"),
             ('later selected unit', f"SELECT id FROM inventory_vehicle_units WHERE id='{max(uuid.UUID(x) for x in body['vehicleIds'])}' FOR UPDATE"),
             ('selected placement', f"SELECT vehicle_id FROM inventory_placements WHERE vehicle_id='{uuid.UUID(f['vehicles'][2])}' FOR UPDATE"),
             ('receipt root', f"SELECT id FROM inventory_receipt_batches WHERE id='{uuid.UUID(b['manualLegacy'])}' FOR UPDATE")]
    for name, statement in locks:
        before = stock_snapshot()
        def attempt():
            started = time.monotonic()
            move(client, w['source'], body, revision, 409, 'stock_busy')
            check(time.monotonic() - started < 2, name + ': NOWAIT exceeded two seconds')
        held_lock(statement, attempt)
        check(before == stock_snapshot(), name + ': contention changed state/facts')
        record('contention_pass', resource=name)


def race(client):
    f = STATE['fixture']
    origin = warehouse(client, 'E2E race origin', 3)
    destination = warehouse(client, 'E2E race one slot', 1)
    bid = receipt(client, origin, quantity=1)['batch']['id']
    body, revision = move_payload(client, origin, destination, [], [(bid, 1)])
    mover, identifier = account('Buyer'), account('Buyer')
    gate = threading.Barrier(2)
    def send(actor, path, payload, rev=None):
        gate.wait(timeout=3)
        return actor.do('POST', path, payload, rev)
    with ThreadPoolExecutor(max_workers=2) as pool:
        moved = pool.submit(send, mover, f'/inventory/warehouses/{origin}/stock-moves', body, revision)
        identified = pool.submit(send, identifier, f'/inventory/receipt-batches/{bid}/identifications',
            {'warehouseId': origin, 'atomic': True, 'items': [{'vin': 'XTAAA11111A990001', 'modelId': f['model']}]})
        responses = [moved.result(), identified.result()]
    check(sum(r['_status'] == 200 for r in responses) == 1, 'Move/identify last-unit race must have exactly one winner')
    check(all(r['_status'] in (200, 409, 412) for r in responses), 'Unexpected contention status')
    after = batch(client, bid)
    check(int(after['batch']['unidentifiedCount']) + int(after['batch']['identifiedCount']) == 1, 'Last-unit conservation failed')
    expected_move_facts = '1' if responses[0]['_status'] == 200 else '0'
    check(rt.sql(STATE, f"SELECT count(*) FROM inventory_facts WHERE receipt_batch_id='{uuid.UUID(bid)}' AND fact_type='receipt.quantity_moved';") == expected_move_facts,
          'Losing movement wrote a stock fact')
    invariants()
    record('race_pass', statuses=[r['_status'] for r in responses], errors=[r.get('error', {}).get('code') for r in responses])


def ui_seed(client):
    f = STATE['fixture']
    source, destination = warehouse(client, 'CUA source warehouse', 40), warehouse(client, 'CUA destination warehouse', 40)
    received = receipt(client, source, vins=['XTAAA11111B700003', 'XTAAA11111B700001', 'XTAAA11111B700004', 'XTAAA11111B700002'])
    pending = [receipt(client, source, quantity=q)['batch']['id'] for q in (5, 4)]
    STATE['uiFixture'] = {'url': rt.UI, 'accountKey': 'Buyer', 'source': source, 'destination': destination,
                          'vehicles': [v['id'] for v in received['vehicles']], 'batches': pending,
                          'vinSorted': sorted(v['vin'] for v in received['vehicles']),
                          'existingOrderId': f['orderId'], 'existingOrderBatch': f['batches']['order'],
                          'existingOrderDestination': f['warehouses']['destination']}
    rt.save(STATE)
    record('cua_fixture_ready', **STATE['uiFixture'])


CASES = {'mixed': mixed, 'rollback': rollback, 'identify': identify, 'correction': correction,
         'contention': contention, 'race': race, 'ui-seed': ui_seed}


def run_checks(selected):
    check(STATE['stage'] in ('current_ready', 'http_in_progress', 'http_failed'), 'Final API must be ready on schema46')
    rt.check_source(STATE)
    check(rt.sql(STATE, 'SELECT version::text || \' \' || dirty::text FROM schema_migrations;') == '46 false', 'Wrong schema for final HTTP checks')
    client = account('Buyer')
    names = [selected] if selected else list(CASES)
    status = STATE.setdefault('httpCases', {})
    for name in names:
        check(name not in status, f'{name} already started; inspect saved evidence and select a fresh affected scenario, never replay successful mutations')
        status[name] = 'running'
        STATE['stage'] = 'http_in_progress'
        rt.save(STATE)
        try:
            CASES[name](client)
        except Exception:
            STATE['stage'] = 'http_failed'
            status[name] = 'failed'
            rt.save(STATE)
            raise
        status[name] = 'passed'
        rt.save(STATE)
        print(f'case={name} passed', flush=True)
    if all(status.get(name) == 'passed' for name in CASES):
        STATE['stage'] = 'http_passed'
        rt.save(STATE)
        print('focused_HTTP_passed; browser coverage remains primary-owned CUA; no release-readiness claim')


def main():
    global STATE, EVENTS
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['seed-legacy', 'check'])
    parser.add_argument('--state', required=True)
    parser.add_argument('--go-ahead', required=True)
    parser.add_argument('--case', choices=list(CASES))
    args = parser.parse_args()
    check(args.go_ahead == rt.GO_AHEAD, 'Explicit final go-ahead required')
    os.umask(0o077)
    STATE = rt.load(args.state)
    rt.container(STATE)
    rt.check_source(STATE)
    evidence = Path(STATE['directory']) / 'http-evidence.json'
    if evidence.exists():
        EVENTS = json.loads(evidence.read_text())
    if args.action == 'seed-legacy':
        check(args.case is None, 'No case selection during seed')
        seed_legacy()
    else:
        run_checks(args.case)


if __name__ == '__main__':
    try:
        main()
    except Exception as exc:
        record('stopped', errorType=type(exc).__name__, reason=str(exc))
        print(f'STOPPED: {type(exc).__name__}: {exc}', file=sys.stderr)
        sys.exit(1)
