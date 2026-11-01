# QA: reproduce, verify, re-check

Modes: `repro` (defect, on `base`), `verify` (run `n`, on the committed fix), `recheck` (run `n` > 1 after a
fix round). The Lead names the run number. Read `references/testbed.md` in full before the first command:
it holds the shared-cluster rules and the measurement methods. Read `cause.md` (cluster scenario) and
`plan.md` (case table, cluster expectations).

Scripts (`lane.sh`, see `_common.md`): `build <work> master|fix` prints the image tag;
`install <work> master|fix <tag>` creates the namespace, the CRDs by scope, the release, the Keycloak CR and
the realm `<slug>-main`; `upgrade <work> <tag>`; `kc <work> ...` for admin calls. They append to
`INVENTORY.txt`; you append everything else you create or change (CRs, secrets, realms, master-realm
objects, ports, `timing: <name> original=<value>`), as you go. Leave tracked files untouched. Keep the setup
installed on pass and on fail; teardown is the Lead's script.

## Mode repro

1. `build master`, then `install master <tag>`: chart, CRDs and image from `src`.
2. Scenario: the CR and the trigger from `cause.md`; confirm by code reading where it fails.
3. Measure with fast feedback: the metrics from `cause.md`.
4. Recovery: the actions from `cause.md`; time to healthy.
5. Save the evidence and `SUMMARY.txt` to `<work>/evidence/master/`.

Report (under 500 words): evidence table, failing step, surprises, the inventory path.

## Mode verify

Defect: reuse the setup. A `crd` item deletes the master-run case CRs first: the run tests the final state
only. `build fix`, `upgrade <tag>`; recreate the scenario; measure the same metrics; expected values from
the case table; run the recovery actions and the spec edit.

Feature: no setup exists. `build fix`, then `install fix <tag>`: chart, CRDs and image from `tree`. `build` and
`install` append to `INVENTORY.txt` below the Lead's "nothing installed" line; append only, never rewrite it. Apply a CR that uses the new field and check
each case-table row; the expected values come from the case table; there is no master measurement.

Run `lane.sh lock show` first and name the live crd items other than yours in the evidence: their CRDs may
be on the cluster.
Save to `<work>/evidence/fix-<n>/`; one table, expected against observed (defect: the master measurement
beside them); mark each case-table row pass or fail. Audit: list every operator write of the run (admin
events, PUT/POST/DELETE in the logs) with the row that explains it; a write no row explains is a failing row.
For each failing row write the observed value, the expected value, and the log line or status field that
shows it: the Developer reads this file without your chat.

Report (under 500 words): the table, recovery times, surprises, the failing rows, timings changed and restored.

## Mode recheck

Run only the rows the Lead names plus the smoke set (one error row, one OK row, one regression row); copy
the other rows from `evidence/fix-<n-1>/` and mark them "carried from run <n-1>". Output as in verify.
