"""Two bounded HTTP operations supporting the primary CUA scenario."""
import importlib.util
import json
from pathlib import Path
import sys

sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location('wh_http', Path(__file__).with_name('http-check.py'))
h = importlib.util.module_from_spec(spec)
spec.loader.exec_module(h)
h.STATE = h.rt.load(sys.argv[2])
h.rt.check_source(h.STATE)
directory = Path(h.STATE['directory'])
h.EVENTS = json.loads((directory / 'http-evidence.json').read_text())
client = h.account('Buyer')
ui = h.STATE['uiFixture']
source, target = ui['source'], ui['destination']
baseline = directory / 'ui-before.json'

if sys.argv[1] == 'bump-target':
    h.check(not baseline.exists(), 'UI target bump is single-use')
    a, b = h.wh(client, source), h.wh(client, target)
    roots = {bid: h.batch(client, bid)['batch'] for bid in ui['batches']}
    changed = client.do('POST', f'/inventory/warehouses/{target}/capacity-changes',
        {'capacity': str(int(b['capacity']) + 1), 'reason': 'Synthetic concurrent update for CUA stale draft'},
        revision=b['revision'], expected=200)['data']
    h.rt.private_write(baseline, {'source': a, 'target': changed, 'roots': roots})
    print('Synthetic destination capacity changed once; stock unchanged. CUA should receive 412 on its captured destination revision.')
elif sys.argv[1] == 'verify':
    before = json.loads(baseline.read_text())
    a, b = h.wh(client, source), h.wh(client, target)
    h.check(int(a['occupied']) == int(before['source']['occupied']) - 5, 'UI source delta')
    h.check(int(b['occupied']) == int(before['target']['occupied']) + 5, 'UI destination delta')
    h.check(int(a['revision']) == int(before['source']['revision']) + 1, 'UI source touched once')
    h.check(int(b['revision']) == int(before['target']['revision']) + 1, 'UI destination touched once')
    stock = client.get(f'/inventory/warehouses/{target}/inventory')['data']
    h.check(sorted(v['vin'] for v in stock['vehicles']) == ui['vinSorted'][:2], 'UI selected VINs differ from committed VINs')
    h.check(sum(int(x['warehouseUnidentifiedCount']) for x in stock['unidentifiedBatches']) == 3, 'UI batch total')
    for bid, root in before['roots'].items():
        current = h.batch(client, bid)['batch']
        h.check(all(current[k] == value for k, value in root.items() if k != 'revision'), 'UI global receipt changed')
    h.invariants()
    result = {'sourceOccupied': a['occupied'], 'destinationOccupied': b['occupied'],
              'movedVINs': ui['vinSorted'][:2], 'unidentifiedMoved': 3,
              'warehouseRevisionsBumpedOnce': True, 'globalReceiptFieldsPreserved': True}
    Path(__file__).with_name('final-ui-http.json').write_text(json.dumps(result, indent=2) + '\n')
    print(json.dumps(result))
else:
    raise RuntimeError('Unknown CUA companion action')
