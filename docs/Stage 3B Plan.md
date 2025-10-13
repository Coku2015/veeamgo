# Stage 3B — Comprehensive UI & Command Adjustments

Source: `smoke_test_logs/UI调整说明.txt` (Stage 3A UI feedback). The items below enumerate every required change for Stage 3B, grouped by command area. Anything marked “Stage 3C” is deferred as noted by the reviewer.

## Proxy & Repository
1. **`proxy states --limit`**
   - Remove this command (table duplicates `proxy list` output).
   - Merge `online`, `disabled`, `outOfDate` columns into `proxy list`.
   - Update docs/help to steer users to `proxy list`.

2. **`proxy list --limit`**
   - Add columns: `Online`, `Disabled`, `Out Of Date` using the state data.
   - Ensure headers are human readable; keep JSON untouched.

3. **`repository get --limit`**
   - Humanize `Type` values (e.g., `Linux Hardened Repository`).
   - Drop columns: `ID`, `Online`, `Out of Date`, `Description`.
   - Keep JSON passthrough and update smoke harness.

4. **`repository describe`**
   - No UI change; ensure docs mention this remains intact.

## Licensing
5. **`license get`**
   - Retain only (in order): `Licensed To`, `Edition`, `Type`, `Status`, `Expires At`, `Support Expires At`.
   - Adjust header capitalization.

6. **`license sockets --limit`**
   - Implement table rendering per OpenAPI schema (workload, host, host ID, type, sockets, cores, expiration if present).
   - Ensure pagination fields populate `result.Raw`.

7. **`license instances --limit`**
   - Use `name` (not `displayName`) for the workload column.
   - Remove `Instance ID` column.
   - Rename headers for readability (`Host`, `Platform`, `Used Instances`, `Revocable`, ...).

8. **`license capacity`**
   - Add table output showing workload, type, used capacity (TB), optional instance ID.
   - Validate via smoke script.

## Options / Configuration
9. **`options get`**
   - Temporarily disable/remove from Stage 3B scope (per note: “not needed now”).
   - Optionally keep available but hidden; update docs accordingly.

10. **`options config-backup` → `configurationbackup describe`**
    - Create new command `veeamgo configurationbackup describe`.
    - Table should mirror “describe” layout with readable headers.
    - Leave old verb as hidden alias for one release; update help/docs.

11. **`options traffic` → `trafficrule get`**
    - Transition to `veeamgo trafficrule get`.
    - Drop the summary table; keep only the traffic rules table with refined headers (`Source Range`, `Encryption`, `Limit Unit`, etc.).

## Exclusion & Service
12. **`exclusion vm list/describe`**
    - Rename to `veeamgo exclusionvm get|describe`.
    - Maintain JSON parity; update CLI help and smoke harness.

13. **`service --limit`**
    - Remove command for now (per instruction: “not needed”).

## Inventory & Protection (Stage 3C)
14. **`inventory servers/objects/physical/physical-items`**
    - Major redesign deferred to Stage 3C; track in backlog.

15. **`protection-group` & `agent` families**
    - Also deferred to Stage 3C for redesign alongside inventory.

## Jobs & Backups
16. **`job get`**
    - Add flag shortcuts for each `EJobType`: `VSphereBackup`, `HyperVBackup`, `VSphereReplica`, `CloudDirectorBackup`, `EntraIDTenantBackup`, `EntraIDAuditLogBackup`, `FileBackupCopy`, `LegacyBackupCopy`, `BackupCopy`, `WindowsAgentBackup`, `LinuxAgentBackup`, `EntraIDTenantBackupCopy`.
    - Remove columns: `Workload`, `Objects`, `High Priority`, `Last Session`, `Progress`.
    - Preserve JSON output.

17. **`job describe`**
    - No change (explicitly accepted).

18. **`backup get` / `backup describe`**
    - Mark as deprecated or hide (UI feedback: “not useful”).
    - Keep APIs for `backup files`; document future removal plan.

19. **`backup files`**
    - Keep command; no UI change required.

## Restore Points & Replica
20. **`restore-point get`**
    - Redesign as `veeamgo restorepoint get --job <name>` (required job context).
    - Add `Backup Job Name` column; remove `Backup ID`, `Session ID`, `ID`.
    - Update header casing; maintain JSON fidelity.

21. **`restore-point describe` & `restore-point disks`**
    - Remove commands (per instruction: “not needed”).

22. **`restore mount get`**
    - Remove command entirely.

23. **`replica get --limit`**
    - Convert to `veeamgo replica get --job <name>` listing replica restore points for a specific job.
    - Add `Replication Job Name`; drop `Job ID`, `Policy Unique ID`, `ID`.
    - Adjust headers accordingly.

24. **`replica describe`**
    - Remove command.

25. **`replica-point get`**
    - Merge into the redesigned `replica get --job` output (see item 23).

26. **`replica-point describe`**
    - Remove command.

## Restore / Replica Session Commands (General)
27. **`restore`-related tables (if any additional “get” commands exist)**
    - Audit the suite to ensure only relevant commands remain per the feedback; remove extraneous ones as needed.

## Tasks & Credentials
28. **`task list/describe/logs`**
    - Remove all three commands.

29. **`credential` suite (`get`, `describe`, `cloud`, `encryption`, `kms`)**
    - Remove from Stage 3B scope (per note: “not needed”).
    - Document removal in changelog/help.

## Security Analyzer & Malware
30. **`security analyzer get`**
    - Remove command (replaced by describe- and send-results-oriented flow).

31. **`security analyzer schedule`**
    - Convert to describe-style single-record output with readable headers.

32. **`security analyzer best-practices`**
    - Rename to `veeamgo security analyzer sendResults`.
    - Remove `ID` column; tidy header casing.

33. **`security events list`, `malware event list`, `malware yara list`**
    - Remove all three commands as requested.

## Smoke Harness & Documentation
34. Update `scripts/veeamgo_e2e.sh` to align with all command renames/removals and new filters.
35. Refresh documentation (`docs/Progress Report.md`, Stage coverage doc, developer guidelines) to mirror the updated CLI surface.
36. Note all removed commands in release notes/changelog; provide migration hints where applicable.

## CLI Verb Placement
37. Restructure command invocation so verbs follow the root (`veeamgo get repository`, `veeamgo describe repository --name <name>`). Update command wiring, help text, and examples, and document the pattern as a Stage 3B principle.

## Validation
38. After implementing all changes:
    - Run `go test ./...`.
    - Execute updated smoke harness.
    - Rebuild release binaries across platforms.
    - Capture sample outputs for QA verification.
