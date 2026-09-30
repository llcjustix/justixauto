# LOCAL-STOP-1 result

Status: DONE

Changed paths: `Makefile`, `README.md`, `tools/stop-apps.py`, and
`tools/test_stop_apps.py`.

`make stop` calls the standard-library helper. It identifies only the current
user's processes with an exact repository or app working directory and a known
local Air/concurrently/API/Vite launch identity, stops supervisors before app
processes, rechecks stable PID/start identity before each signal, and reports occupied
ports without using a port as signal authorization. It uses a 20-second graceful
ceiling, then SIGKILL only for still-verified owned survivors. The helper reads
only a simple `HTTP_ADDR` assignment from `.env`; inherited `HTTP_ADDR` wins,
and the fallback is `127.0.0.1:8080`.

Unit evidence (mocked process, signal, time, and listener fixtures only):

```text
$ PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tools -p 'test_stop_apps.py'
..........
----------------------------------------------------------------------
Ran 10 tests in 0.006s

OK
```

Limitations: OS process inspection and signaling are separate operations, so
identity rechecks reduce but cannot eliminate PID-reuse races. Unsupported
`HTTP_ADDR` shell syntax fails closed. No live stop, port binding, process
signal, database action, build, lint, typecheck, browser, or integration check
was executed; this evidence does not claim live port release or release readiness.
