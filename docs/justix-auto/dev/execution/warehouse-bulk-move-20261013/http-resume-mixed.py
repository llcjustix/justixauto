#!/usr/bin/env python3
"""Resume only the recorded replay failure; never repeat the initial transfer."""
import argparse
import importlib.util
import json
from pathlib import Path
import sys

sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location('warehouse_http', Path(__file__).with_name('http-check.py'))
h = importlib.util.module_from_spec(spec)
spec.loader.exec_module(h)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--state', required=True)
    parser.add_argument('--go-ahead', required=True)
    args = parser.parse_args()
    h.check(args.go_ahead == 'warehouse-replay-continuation', 'Exact affected continuation authorization required')
    h.STATE = h.rt.load(args.state)
    h.rt.check_source(h.STATE)
    h.rt.container(h.STATE)
    h.check(h.STATE['stage'] == 'http_failed' and h.STATE['httpCases'] == {'mixed': 'failed'}, 'Fixture is not the untouched failed mixed scenario')
    h.EVENTS = json.loads((Path(h.STATE['directory']) / 'http-evidence.json').read_text())
    request = h.STATE['fixture']['mixedRequest']
    path = '/inventory/warehouses/' + request['source'] + '/stock-moves'
    prior = [e for e in h.EVENTS if e.get('method') == 'POST' and e.get('path') == path]
    h.check([e['status'] for e in prior] == [200, 409] and prior[1]['error'] == 'placement_changed',
            'Expected exactly initial success and failed replay, with no later source mutation')
    h.record('resume_initial_provenance', initialTransferAt=prior[0]['at'], originalReplayAt=prior[1]['at'],
             originalError='placement_changed', initialTransferRepeated=False,
             sourceRebindings=h.STATE.get('sourceRebindings', []))
    client = h.account('Buyer')
    h.reject('repaired exact replay', client, request['source'], request['body'], request['revision'], 412, 'stale_revision')
    f, w, b = h.STATE['fixture'], h.STATE['fixture']['warehouses'], h.STATE['fixture']['batches']
    onwards, revision = h.move_payload(client, w['destination'], w['third'], [], [(b['manualLegacy'], 1)])
    h.move(client, w['destination'], onwards, revision)
    single = client.do('POST', f'/inventory/vehicle-units/{f["vehicles"][0]}/warehouse-moves',
        {'fromWarehouseId': w['destination'], 'toWarehouseId': w['third'], 'occurredAt': h.now()}, expected=200)['data']
    h.check(single['placement']['receiptBatchId'] == b['identified'], 'Legacy single-move receipt provenance lost')
    h.invariants()
    h.STATE['httpCases']['mixed'] = 'passed'
    h.STATE['stage'] = 'http_in_progress'
    h.rt.save(h.STATE)
    h.record('mixed_continuation_pass', replayStatus=412, replayUnchanged=True,
             quantityOnward=True, legacyMoveProvenance=True, initialTransferRepeated=False)
    print('mixed_continuation=passed; initial transfer not repeated; original failure evidence preserved')


if __name__ == '__main__':
    try:
        main()
    except Exception as exc:
        h.record('continuation_stopped', errorType=type(exc).__name__, reason=str(exc))
        print(f'CONTINUATION_STOPPED: {type(exc).__name__}: {exc}', file=sys.stderr)
        sys.exit(1)
