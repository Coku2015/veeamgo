# UI Design Reference Samples

[TOC]



All examples below show the expected CLI experience for the read-only command set. Every snippet uses the generic shell prompt `user@host %`. Sample payloads are based on the OpenAPI examples and deterministic lab data so screenshots can be regenerated consistently.

---

## Server Commands

### `veeamgo get server info`

**Help**
```
user@host % veeamgo get server info --help
Show server metadata

Usage:
  veeamgo get server info [flags]

Flags:
  -h, --help   help for info

Global Flags:
      --config string   Path to configuration file
      --output string   Output format (table|json) (default "table")
      --profile string  Profile name to use
```

**Success**
```
user@host % veeamgo get server info
Name:              vbrsav13.backupnext.home
Platform:          Linux
VBR uuid:          99ab520d-aa2c-4189-b908-61557da58330
Build Version:     13.0.0.4967
Database:          PostgreSQL
Database Version:  PostgreSQL 17.6 on x86_64-pc-linux-gnu, compiled by gcc (GCC) 11.5.0 20240719 (Red Hat 11.5.0-5), 64-bit
Database Edition:
Patches:
```

**Error**
```
user@host % veeamgo get server info --unknown
Error: unknown flag: --unknown
Usage:
  veeamgo get server info [flags]
```

### `veeamgo get server time`

**Help**
```
user@host % veeamgo get server time --help
Show server clock information

Usage:
  veeamgo get server time [flags]

Flags:
  -h, --help   help for time

Global Flags:
      --config string   Path to configuration file
      --output string   Output format (table|json) (default "table")
      --profile string  Profile name to use
```

**Success**
```
user@host % veeamgo get server time
Server Time:  2025-10-11 13:09:10 (+08:00)
Timezone:     (UTC+08:00) China Standard Time (Shanghai)
```

**Error**
```
user@host % veeamgo get server time extra
Error: unknown command "extra" for "veeamgo get server time"
Run 'veeamgo get server time --help' for usage.
```

### `veeamgo get managedserver`

**Help**
```
user@host % veeamgo get managedserver --help
List managed servers

Usage:
  veeamgo get managedserver [flags]

Flags:
      --limit int     Maximum number of managed servers to return (default: all)
      --name string   Filter by name pattern (supports * wildcards)
      --type strings  Filter by managed server type (e.g. WindowsHost, LinuxHost)
  -h, --help          help for managedserver

Global Flags:
      --config string   Path to configuration file
      --output string   Output format (table|json) (default "table")
      --profile string  Profile name to use
```

**Success**
```
user@host % veeamgo get managedserver --limit 2
Name                Type              Status     Description                                      Id
srv82.tech.local    Microsoft Windows Available  Windows managed server with deployment kit installed  3c9f0bc9-07c3-4bc1-bf2b-9bc445dbb5d2
linrepo01.tech.local  Linux             Available  Linux gateway for hardened repositories             4d53f576-06cb-4c01-9a20-1f6333c5805d
```

**Error**
```
user@host % veeamgo get managedserver --type FooType
Error: invalid managed server type "FooType"
Usage:
  veeamgo get managedserver [flags]
```

### `veeamgo describe managedserver`

**Help**

```
user@host % veeamgo describe managedserver --help
Show detailed managed server information

Usage:
  veeamgo describe managedserver [flags]

Flags:
      --id string     Managed server identifier
  -h, --help   help for managedserver

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo describe managedserver --id b01df8b9-4106-41fc-a64b-109ae4d62784
Name:        srv82.tech.local
Type:        WindowsHost
Status:      Available
Description: Windows managed server with deployment kit installed
Id:          b01df8b9-4106-41fc-a64b-109ae4d62784
```

**Error**
```
user@host % veeamgo describe managedserver
Error: specify --id <managed-server-id>
```

### `veeamgo add managedserver vsphere`

**Help**
```
user@host % veeamgo add managedserver vsphere --help
Registers a VMware vSphere server (vCenter or ESXi) and optionally provisions credentials automatically.

Usage:
  veeamgo add managedserver vsphere [flags]

Flags:
      --credential-description string   Description for created credentials (auto-generated if omitted)
      --credentials-id string           Existing credentials ID to reuse
      --description string              Description for the managed server (auto-generated if omitted)
  -h, --help                            help for vsphere
      --name string                     vSphere server DNS name or IP address
      --password string                 Password for creating credentials (omit to prompt securely)
      --port int                        Port used to communicate with the vSphere server (default 443)
      --thumbprint string               TLS thumbprint used to validate the server identity
      --username string                 Username for creating credentials (mutually exclusive with --credentials-id)
      --wait                            Wait for the provisioning session to complete
      --yes                             Confirm without prompting (required for non-interactive use)

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success (`--wait`)**
```
user@host % veeamgo add managedserver vsphere --name vc01.lab.local --username svc-vbr --password 'S3cret!' --wait --yes
Created credentials "VMware credentials for vc01.lab.local (svc-vbr, created 2025-10-17)" (5d9d8ef1-45c0-4f29-8bc4-9f735b1079c2).
VMware managed server "vc01.lab.local" provisioning finished with result Success (session 1f74e9ab-8e51-4b0d-92e3-62c63c9b9183).
Managed server ID: 3c9f0bc9-07c3-4bc1-bf2b-9bc445dbb5d2
```

**Success (async)**
```
user@host % veeamgo add managedserver vsphere --name esxi01.lab.local --username svc-vbr --password 'S3cret!' --yes
Created credentials "VMware credentials for esxi01.lab.local (svc-vbr, created 2025-10-17)" (2b8ca7c9-f2d7-41fd-a931-1f3f5c1f5bd2).
Started provisioning VMware managed server "esxi01.lab.local" (session 5b4ec79a-0d5f-41df-9271-dc5c4f6f2c1f).
Use veeamgo describe session --id 5b4ec79a-0d5f-41df-9271-dc5c4f6f2c1f to track progress.
```

### `veeamgo add managedserver windows`

**Help**
```
user@host % veeamgo add managedserver windows --help
Registers a Windows server and can create standard credentials automatically or rely on certificate-based authentication when the deployment kit is installed.

Usage:
  veeamgo add managedserver windows [flags]

Flags:
      --credential-description string   Description for created credentials (auto-generated if omitted)
      --credentials-id string           Existing credentials ID to reuse
      --connect-mode string             Connection mode (Credential or Certificate) (default "Credential")
      --description string              Description for the managed server (auto-generated if omitted)
  -h, --help                            help for windows
      --name string                     Windows server DNS name or IP address
      --password string                 Password for creating credentials (omit to prompt securely)
      --username string                 Username for creating credentials (mutually exclusive with --credentials-id)
      --wait                            Wait for the provisioning session to complete
      --yes                             Confirm without prompting (required for non-interactive use)

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success (`--wait`)**
```
user@host % veeamgo add managedserver windows --name winrepo01.lab.local --username administrator --password 'S3cret!' --wait --yes
Created credentials "Microsoft Windows credentials for winrepo01.lab.local (administrator, created 2025-10-17)" (c3a1fd8b-58d0-4c9b-a0ef-3ab8f3df9e34).
Microsoft Windows managed server "winrepo01.lab.local" provisioning finished with result Success (session 0d0fbf19-6b3f-4fb3-b889-75d0093b5a0c).
Managed server ID: 4d53f576-06cb-4c01-9a20-1f6333c5805d
```

**Async reuse**
```
user@host % veeamgo add managedserver windows --name winrepo02.lab.local --credentials-id c3a1fd8b-58d0-4c9b-a0ef-3ab8f3df9e34 --yes
Started provisioning Microsoft Windows managed server "winrepo02.lab.local" (session 96b2c1f5-8d91-4f24-9b1a-d84c9dc8693b).
Use veeamgo describe session --id 96b2c1f5-8d91-4f24-9b1a-d84c9dc8693b to track progress.
```

**Certificate (`--wait`)**
```
user@host % veeamgo add managedserver windows --name winrepo-cert.lab.local --connect-mode Certificate --wait --yes
Microsoft Windows managed server "winrepo-cert.lab.local" provisioning finished with result Success (session 7d8c25c1-4a1d-440a-9d7c-1e76a971ab6f).
Managed server ID: 9fa0dd63-0f61-4e5c-a985-2a78c5df45ab
```

### `veeamgo add managedserver linux`

**Help**
```
user@host % veeamgo add managedserver linux --help
Registers a Linux managed server. Supports permanent credentials, single-use SSH credentials, or certificate-based pairing when the deployment kit is installed.

Usage:
  veeamgo add managedserver linux [flags]

Flags:
      --connect-mode string             Connection mode (Credential, SingleUse, or Certificate) (default "Credential")
      --credential-description string   Description for created credentials (auto-generated if omitted)
      --credentials-id string           Existing credentials ID to reuse
      --description string              Description for the managed server (auto-generated if omitted)
      --handshake-code string           Handshake code for certificate-based pairing (optional)
  -h, --help                            help for linux
      --name string                     Linux server DNS name or IP address
      --password string                 Password for creating credentials (omit to prompt securely)
      --single-use-add-sudo             Automatically add account to sudoers in single-use mode
      --single-use-auth-type string     Authentication type for single-use credentials (Password, PrivateKey, or Auto)
      --single-use-elevate              Elevate permissions to root for single-use credentials
      --single-use-passphrase string    Passphrase protecting the private key
      --single-use-password string      Password for single-use credentials
      --single-use-private-key string   Private key for single-use credentials (PEM)
      --single-use-root-password string Root password used when elevating privileges
      --single-use-ssh-port int         SSH port for single-use credentials (default 22)
      --single-use-use-su               Use su instead of sudo in single-use mode
      --single-use-username string      Username for single-use credentials (SingleUse mode)
      --ssh-fingerprint string          SSH fingerprint used to verify the server identity
      --username string                 Username for creating credentials (Credential mode)
      --wait                            Wait for the provisioning session to complete
      --yes                             Confirm without prompting (required for non-interactive use)

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success (`--wait`, credential mode)**
```
user@host % veeamgo add managedserver linux --name repo01.lab.local --ssh-fingerprint "ssh-rsa 3072 AAAA..." --username veeam --password 'S3cret!' --wait --yes
Created credentials "Linux credentials for repo01.lab.local (veeam, created 2025-10-17)" (7a5fe216-4980-4ae8-845e-3f9ff88c1234).
Linux managed server "repo01.lab.local" provisioning finished with result Success (session 10637a4f-5ac0-440a-9700-2921a29b46bc).
Managed server ID: e2e1c7c5-4389-4d0e-9816-8f83f6c98c42
```

**Success (`--wait`, single-use mode)**
```
user@host % veeamgo add managedserver linux \
    --name repo02.lab.local \
    --ssh-fingerprint "ssh-rsa 3072 AAAA..." \
    --connect-mode SingleUse \
    --single-use-username repo \
    --single-use-password 'S3cret!' \
    --single-use-elevate \
    --wait --yes
Linux managed server "repo02.lab.local" provisioning finished with result Success (session 3bd08254-3b6f-4220-9410-cb7d19fdb7cf).
Managed server ID: 0c8d2af4-7922-4e02-bf40-d9c0c59c5d5f
```

---

## Repository Commands

### `veeamgo get repository`

**Help**
```
user@host % veeamgo get repository --help
List backup repositories

Usage:
  veeamgo get repository [flags]

Flags:
      --limit int     Maximum number of repositories to return (default: all)
      --name string   Filter by repository name (supports * wildcards)
      --type strings  Filter by repository type (e.g. WinLocal, LinuxLocal)
  -h, --help          help for repository

Global Flags:
      --config string   Path to configuration file
      --output string   Output format (table|json) (default "table")
      --profile string  Profile name to use
```

**Success**
```
user@host % veeamgo get repository --limit 2
Name                       Type           Host                      Path                                  Capacity(GB)  Free(GB)  Used(GB)  Online  Out of Date  Description                                Id
Default Backup Repository  LinuxHardened  vbrsav13.backupnext.home  /var/lib/veeam/backup                 260.9         259       0         Yes     No           Created by Veeam Backup                    88788f9e-d8f5-4eb4-bc4f-9b3f5403bcec
Synology NFS               Nfs            Gateway (auto)            nfs://10.10.1.45:/volume1/VBRBackups  50042.3       8878.7    6878.9    Yes     No           Created by .\\veeamadmin at 2025/9/7 8:43.  526cc0d7-5a43-4f41-b491-1efaa151f492
```

**Error**
```
user@host % veeamgo get repository --type FooRepo
Error: invalid repository type "FooRepo"
Usage:
  veeamgo get repository [flags]
```

### `veeamgo describe repository`

**Help**
```
user@host % veeamgo describe repository --help
Show detailed repository configuration

Usage:
  veeamgo describe repository [flags]

Flags:
      --name string   Repository name to describe
  -h, --help          help for repository
```

**Success**

```
user@host % veeamgo describe repository --name "Default Backup Repository"
Name:          Default Backup Repository
Type:          LinuxHardened
Description:   Created by Veeam Backup
Host:          vbrsav13.backupnext.home
Path:          /var/lib/veeam/backup
Capacity(GB):  260.9
Free(GB):      259
Used(GB):      0
Online:        Yes
Out of Date:   No
Id:            88788f9e-d8f5-4eb4-bc4f-9b3f5403bcec
Config:
  hostId: vbrsav13.backupnext.home
  path: /var/lib/veeam/backup
  type: LinuxHardened
  ...
```

**Error**
```
user@host % veeamgo describe repository
Error: flag --name is required (provide the repository name)
Usage:
  veeamgo describe repository [flags]
```

---

## Scale-out Repository Commands

### `veeamgo get sobr`

**Help**
```
user@host % veeamgo get sobr --help
List scale-out backup repositories

Usage:
  veeamgo get sobr [flags]

Flags:
      --limit int     Maximum number of scale-out repositories to return (default: all)
      --name string   Filter by scale-out repository name (supports * wildcards)
  -h, --help          help for sobr
```

**Success**
```
user@host % veeamgo get sobr --limit 2
Name           Performance Extents  Placement Policy  Strict Placement  Capacity Tier                        Archive Tier
Primary SOBR   3                    Data Locality     Yes               Enabled | 3 extent(s) | Copy & Move   Enabled | Extent arch-1 | After 30 day(s)
Archive SOBR   2                    Performance       No                Disabled                             Disabled
```

**Error**
```
user@host % veeamgo get sobr --sort foo
Error: invalid orderColumn "foo"
Usage:
  veeamgo get sobr [flags]
```

### `veeamgo describe sobr`

**Help**
```
user@host % veeamgo describe sobr --help
Show detailed scale-out repository configuration

Usage:
  veeamgo describe sobr [flags]

Flags:
  -h, --help          help for sobr
      --name string   Scale-out repository name to describe

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo describe sobr --name "Primary SOBR"
Name:                          Primary SOBR
Description:                   Consolidated repository for production jobs
Id:                            98cc7718-2bb3-47b9-9d47-7e3ad0e6d2aa
Unique Id:                     SOBR://0ab32c2e-3eaa-4a5c-993f-03cca6de3281
Placement Policy:              Data Locality
Strict Placement:              Yes
Per-VM Backup:                 Yes
Full Backup When Extent Offline:  No
Performance Extents:
  Repo1 (1f2f4b9d-6f4d-4f10-af1e-3c7c90b2aa24) [Normal]
  Repo2 (447f2c94-0c54-4a2c-9d8d-4a3d5e1d6fd9) [Normal, Maintenance]
  Repo3 (5c547868-1350-4b77-a19d-77cb1b6200a9) [Normal]
Capacity Tier:
  Enabled | 3 extent(s) | Copy & Move
Archive Tier:
  Enabled | Extent arch-1 | After 30 day(s)
Config:
  performanceTier:
    performanceExtents:
      - id: 1f2f4b9d-6f4d-4f10-af1e-3c7c90b2aa24
        name: Repo1
        status:
          - Normal
      - id: 447f2c94-0c54-4a2c-9d8d-4a3d5e1d6fd9
        name: Repo2
        status:
          - Normal
          - Maintenance
      - id: 5c547868-1350-4b77-a19d-77cb1b6200a9
        name: Repo3
        status:
          - Normal
  capacityTier:
    isEnabled: true
    copyPolicyEnabled: true
    movePolicyEnabled: true
    extents:
      - id: cap-01
      - id: cap-02
      - id: cap-03
  archiveTier:
    isEnabled: true
    extentId: arch-1
    archivePeriodDays: 30
```

**Error**
```
user@host % veeamgo describe sobr
Error: flag --name is required (provide the scale-out repository name)
Usage:
  veeamgo describe sobr [flags]
```

---

## WAN Accelerator Commands

### `veeamgo get wanaccelerator`

**Help**
```
user@host % veeamgo get wanaccelerator --help
List WAN accelerators

Usage:
  veeamgo get wanaccelerator [flags]

Flags:
      --desc          Sort results in descending order
      --limit int     Maximum number of WAN accelerators to return (default: all)
      --name string   Filter by WAN accelerator name (supports * wildcards)
      --sort string   Sort results by column (name|hostId|trafficPort)
  -h, --help          help for wanaccelerator
```

**Success**
```
user@host % veeamgo get wanaccelerator --limit 2
Name                       Description                    Traffic Port  Streams  High Bandwidth Mode  Cache Folder  Cache Size
enterprise03.tech.local    Created by TECH\sheila.d.cory  6165          5        Yes                  C:\VeeamWAN  100 GB
enterprise05.tech.local    Created by TECH\sheila.d.cory  6165          5        No                   C:\VeeamWAN  50 GB
```

**Error**
```
user@host % veeamgo get wanaccelerator --sort foo
Error: invalid orderColumn "foo"
```

### `veeamgo describe wanaccelerator`

**Help**
```
user@host % veeamgo describe wanaccelerator --help
Show detailed WAN accelerator configuration

Usage:
  veeamgo describe wanaccelerator [flags]

Flags:
  -h, --help          help for wanaccelerator
      --name string   WAN accelerator name to describe

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo describe wanaccelerator --name "enterprise03.tech.local"
Name:                enterprise03.tech.local
Description:         Created by TECH\sheila.d.cory
Id:                  d8ecb099-ca8e-4cd3-b12f-10f36508942b
Host Id:             b29b8591-bf31-4174-b69b-25ac296a20b2
Traffic Port:        6165
Streams:             5
High Bandwidth Mode: Yes
Cache Folder:        C:\VeeamWAN
Cache Size:          100 GB
Config:
  cache:
    cacheFolder: C:\VeeamWAN
    cacheSize: 100
    cacheSizeUnit: GB
  id: d8ecb099-ca8e-4cd3-b12f-10f36508942b
  name: enterprise03.tech.local
  server:
    description: Created by TECH\sheila.d.cory
    highBandwidthModeEnabled: true
    hostId: b29b8591-bf31-4174-b69b-25ac296a20b2
    streamsCount: 5
    trafficPort: 6165
```

**Error**
```
user@host % veeamgo describe wanaccelerator
Error: flag --name is required (provide the WAN accelerator name)
Usage:
  veeamgo describe wanaccelerator [flags]
```

---

## Global Options

### `veeamgo get generaloption`

**Help**
```
user@host % veeamgo get generaloption --help
Show global notification and SIEM options

Usage:
  veeamgo get generaloption [flags]

Flags:
  -h, --help   help for generaloption

Global Flags:
      --config string   Path to configuration file
      --output string   Output format (table|json) (default "table")
      --profile string  Profile name to use
```

**Success**
```
user@host % veeamgo get generaloption
Notifications Enabled:            Yes
Storage Space Threshold Alert:    Yes
Datastore Space Threshold Alert:  No
Skip VM Space Threshold Alert:    Yes
Support Expiration Alerts:        Yes
Update Notifications:             No
SIEM SNMP Events:                 Yes
SIEM Syslog Events:               No
Config:
  notificationEnabled: true
  notifications:
    datastoreSpaceThresholdEnabled: false
    notifyOnSupportExpiration: true
    notifyOnUpdates: false
    skipVMSpaceThresholdEnabled: true
    storageSpaceThresholdEnabled: true
  siemIntegration:
    SNMPEventsEnabled: true
    SyslogEventsEnabled: false
```

**Error**
```
user@host % veeamgo get generaloption --unknown
Error: unknown flag: --unknown
```

---

## Security & Malware Commands

### `veeamgo get malwaredetectionevent`

**Help**
```
user@host % veeamgo get malwaredetectionevent --help
List malware detection events

Usage:
  veeamgo get malwaredetectionevent [flags]

Flags:
      --backup-object string   Filter by backup object ID
      --created-by string      Filter by event creator
      --desc                   Sort results in descending order
      --detected-before string Include events detected on/before RFC3339 timestamp
      --detected-since string  Include events detected on/after RFC3339 timestamp
      --engine string          Filter by detection engine
      --limit int              Maximum number of events to return (default: all)
      --machine-name string    Filter by machine display name
      --severity string        Filter by severity (Clean|Suspicious|Infected|Informative)
      --sort string            Sort results by column (e.g. detectionTimeUtc)
      --source string          Filter by source (Manual|InternalVeeamDetector|External|MarkAsCleanEvent)
      --state string           Filter by event state (Created|FalsePositive)
      --type string            Filter by detection type (e.g. YaraScan, AntivirusScan)
  -h, --help                   help for malwaredetectionevent

Global Flags:
      --config string   Path to configuration file
      --output string   Output format (table|json) (default "table")
      --profile string  Profile name to use
```

**Success**
```
user@host % veeamgo get malwaredetectionevent --limit 2
Event ID                               Type            Severity     State    Source                   Engine    Machine  Detected At               Created At                Created By  Details
0d6e8fe3-7d4c-4e8c-9b16-934d423737a1    YaraScan        Suspicious   Created  InternalVeeamDetector   Yara      srv01    2025-10-10 18:41:07 (+08:00)  2025-10-10 18:41:05 (+08:00)  sensor     Rule hit: ransomware.yar
9bf7089b-0f4e-494f-ae3f-3c4d25f7f5dd    AntivirusScan   Infected     Created  External                Defender  srv02    2025-10-09 22:13:52 (+08:00)  2025-10-09 22:13:48 (+08:00)  soc        Malware family: Trojan.Generic
```

**Error**
```
user@host % veeamgo get malwaredetectionevent --limit -1
Error: invalid value "-1" for flag --limit: parse error
```

### `veeamgo get yararule`

**Help**
```
user@host % veeamgo get yararule --help
List uploaded YARA rules

Usage:
  veeamgo get yararule [flags]

Flags:
  -h, --help   help for yararule

Global Flags:
      --config string   Path to configuration file
      --output string   Output format (table|json) (default "table")
      --profile string  Profile name to use
```

**Success**

```
user@host % veeamgo get yararule
File Name
ransomware.yar
coinminer.yar
```

**Error**
```
user@host % veeamgo get yararule --unknown
Error: unknown flag: --unknown
```

---

## Job Commands

### `veeamgo get job`

**Help**
```
user@host % veeamgo get job --help
List jobs with current state

Usage:
  veeamgo get job [flags]

Flags:
      --after-job string   Only include jobs chained after the specified job name
      --before string      Only include jobs with last run ≤ RFC3339 timestamp
      --high-priority      Only include jobs marked high priority
      --limit int          Maximum number of jobs to return (default: all)
      --name string        Filter by job name (supports * wildcards)
      --repository string  Filter by target repository name
      --result string      Filter by last result (e.g. Success, Warning, Failed)
      --since string       Only include jobs with last run ≥ RFC3339 timestamp
      --status string      Filter by current job status (e.g. Running, Stopped)
      --type strings       Filter by job type (repeatable; accepts EJobType values)
      --workload string    Filter by workload (e.g. Vmware, HyperV)
  -h, --help               help for job

Global Flags:
      --config string   Path to configuration file
      --output string   Output format (table|json) (default "table")
      --profile string  Profile name to use
```

**Success**
```
user@host % veeamgo get job --limit 1
Name          Type           Workload  Status   Last Result  Last Run              Next Run              Repository  Objects  High Priority  Last Session                          Progress
Backup Job 1  VSphereBackup  vm        stopped  Success      2025-10-10T14:00:12Z  2025-10-11T14:00:00Z  VHR-01      2        No             199e3af7-2543-4eb8-b4f5-ae0ab615c386  100%
```

**Error**
```
user@host % veeamgo get job --type Foo
Error: invalid job type "Foo"
```

### `veeamgo describe job`

**Help**
```
user@host % veeamgo describe job --help
Show detailed job configuration and state

Supported job types:
  - VMware Backup
  - Hyper-V Backup
  - VMware Replication
  - Cloud Director Backup
  - Microsoft Entra Tenant Backup
  - Microsoft Entra Audit Log Backup
  - File Backup Copy
  - Legacy Backup Copy
  - Backup Copy
  - Windows Agent Backup
  - Linux Agent Backup
  - Microsoft Entra Tenant Backup Copy
  - NAS Backup
  - NAS Backup Copy
  - Oracle RMAN Backup
  - SAP HANA Backup
  - AWS Backup Copy

Usage:
  veeamgo describe job <name> [flags]

Flags:
  -h, --help   help for job

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo describe job "Backup Job 1"
Name:             Backup Job 1
Type:             VSphereBackup
Workload:         vm
Description:      Created by .\veeamadmin at 2025/09/16 12:57 PM
Status:           stopped
Last Result:      Success
Last Run:         2025-10-10T14:00:12Z
Next Run:         2025-10-11T14:00:00Z
Next Run Policy:  10/11/2025 10:00 PM
Repository:       VHR-01 (47f8a73a-30cf-46a9-b59d-e5df16ccbf91)
Objects:          2
High Priority:    No
Last Session:     199e3af7-2543-4eb8-b4f5-ae0ab615c386
Run After:
Disabled:         No
Config:
  id: e4bf1214-0d25-420d-8419-0643d9b90eef
  name: Backup Job 1
  schedule:
    daily:
      isEnabled: true
      localTime: "22:00"
    runAutomatically: true
  storage:
    backupRepositoryId: 47f8a73a-30cf-46a9-b59d-e5df16ccbf91
    backupModeType: Incremental
  virtualMachines:
    includes:
      - name: ubuntu
        objectId: vm-16
        platform: VSphere
      - name: Win001
        objectId: vm-15
```

**Error** 

```
user@host % veeamgo describe job
Error: accepts 1 arg(s), received 0
Usage:
  veeamgo describe job <name> [flags]
```

---

### `veeamgo clone job`

**Help**
```
user@host % veeamgo clone job --help
Clone a job

Usage:
  veeamgo clone job [flags]

Flags:
      --id string    Job ID to target (optional)
      --name string  Job name to target (optional)
      --yes          Confirm without prompting (required to execute)
  -h, --help         help for job

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo clone job --name "Backup Job 1" --yes
Cloned job "Backup Job 1" (2f4a2f68-6ec6-48f5-bd6f-1e2a0af7ca0d) to "Backup Job 1 - Copy" (7a90fd77-5e2d-4e75-9b3f-6d0f69ad8f42) [VMware Backup].
```

**Error**
```
user@host % veeamgo clone job --yes
Error: provide --name or --id
```

### `veeamgo delete job`

**Help**
```
user@host % veeamgo delete job --help
Delete a job

Usage:
  veeamgo delete job [flags]

Flags:
      --id string    Job ID to target (optional)
      --name string  Job name to target (optional)
      --yes          Confirm without prompting (required to execute)
  -h, --help         help for job

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo delete job --name "Backup Job 1" --yes
Deleted job "Backup Job 1" (2f4a2f68-6ec6-48f5-bd6f-1e2a0af7ca0d).
```

**Error**
```
user@host % veeamgo delete job --name "Backup Job 1" --id 2f4a2f68-6ec6-48f5-bd6f-1e2a0af7ca0d --yes
Error: provide either --name or --id, not both
```

### `veeamgo enable job`

**Help**
```
user@host % veeamgo enable job --help
Enable a job

Usage:
  veeamgo enable job [flags]

Flags:
      --id string    Job ID to target (optional)
      --name string  Job name to target (optional)
      --yes          Confirm without prompting (required to execute)
  -h, --help         help for job

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo enable job --name "Backup Job 1" --yes
Enabled job "Backup Job 1" (2f4a2f68-6ec6-48f5-bd6f-1e2a0af7ca0d).
```

**Error**
```
user@host % veeamgo enable job --yes
Error: provide --name or --id
```

### `veeamgo disable job`

**Help**
```
user@host % veeamgo disable job --help
Disable a job

Usage:
  veeamgo disable job [flags]

Flags:
      --id string    Job ID to target (optional)
      --name string  Job name to target (optional)
      --yes          Confirm without prompting (required to execute)
  -h, --help         help for job

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo disable job --name "Backup Job 1" --yes
Disabled job "Backup Job 1" (2f4a2f68-6ec6-48f5-bd6f-1e2a0af7ca0d).
```

**Error**
```
user@host % veeamgo disable job --yes
Error: provide --name or --id
```

### `veeamgo start job`

**Help**
```
user@host % veeamgo start job --help
Start a job

Usage:
  veeamgo start job [flags]

Flags:
      --active-full                Perform an active full backup
      --id string                  Job ID to target (optional)
      --name string                Job name to target (optional)
      --start-chained              Start chained jobs as well
      --sync-restore-points string   For backup copy jobs: sync restore points (All|Latest)
      --yes                        Confirm without prompting (required to execute)
  -h, --help                       help for job

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo start job --name "Backup Job 1" --yes
Started job "Backup Job 1" (2f4a2f68-6ec6-48f5-bd6f-1e2a0af7ca0d). Session d5c9f1a6-1954-4f82-9ad0-6e3fba0fd0d4 (Running).
```

**Error**
```
user@host % veeamgo start job --yes
Error: provide --name or --id
```

### `veeamgo stop job`

**Help**
```
user@host % veeamgo stop job --help
Stop a job

Usage:
  veeamgo stop job [flags]

Flags:
      --cancel-chained   Cancel chained jobs as well
      --graceful         Perform a graceful stop (default true)
      --id string        Job ID to target (optional)
      --name string      Job name to target (optional)
      --yes              Confirm without prompting (required to execute)
  -h, --help             help for job

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo stop job --name "Backup Job 1" --yes
Initiated stop for job "Backup Job 1" (2f4a2f68-6ec6-48f5-bd6f-1e2a0af7ca0d). Session 79a6ad69-7bfd-4d97-90d7-b70488e6af92 (Stopping).
```

**Error**
```
user@host % veeamgo stop job --yes
Error: provide --name or --id
```

### `veeamgo retry job`

**Help**
```
user@host % veeamgo retry job --help
Retry a job

Usage:
  veeamgo retry job [flags]

Flags:
      --id string          Job ID to target (optional)
      --name string        Job name to target (optional)
      --start-chained      Start chained jobs as well
      --yes                Confirm without prompting (required to execute)
  -h, --help               help for job

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo retry job --name "Backup Job 1" --yes
Retry initiated for job "Backup Job 1" (2f4a2f68-6ec6-48f5-bd6f-1e2a0af7ca0d). Session 3ed16c1f-9347-44cb-a63f-9a0a2f0aaea1 (Running).
```

**Error**
```
user@host % veeamgo retry job --yes
Error: provide --name or --id
```

---

## Session Commands

Session和Task子命令还得重新调整下。

### `veeamgo get session`

**Help**
```
user@host % veeamgo get session --help
List Veeam job sessions

Usage:
  veeamgo get session [flags]

Flags:
      --created-before string   Include sessions created on/before RFC3339 timestamp
      --created-since string    Include sessions created on/after RFC3339 timestamp
      --desc                    Sort in descending order (default ascending)
      --ended-before string     Include sessions ended on/before RFC3339 timestamp
      --ended-since string      Include sessions ended on/after RFC3339 timestamp
      --job string              Filter by job ID
      --limit int               Maximum number of sessions to return (default: all)
      --name string             Filter by session name (supports * wildcards)
      --result strings          Filter by session result
      --sort string             Sort sessions by column (e.g. creationTime, endTime)
      --state strings           Filter by session state
      --type strings            Filter by session type (repeatable)
  -h, --help                    help for session

Global Flags:
      --config string   Path to configuration file
      --output string   Output format (table|json) (default "table")
      --profile string  Profile name to use
```

**Success**
```
user@host % veeamgo get session --limit 1
Name                      Type                  State    Result   Progress  Created                 Ended                   Session ID
Backup Configuration Job  ConfigurationBackup   Stopped  Success  100%      2023-11-08 10:00:12 (+01:00)  2023-11-08 10:00:34 (+01:00)  f848e90c-7f37-4ff5-9d55-04e33f8a4de3
```

**Error**
```
user@host % veeamgo get session --created-since not-a-time
Error: invalid time "not-a-time" (use RFC3339)
```

### `veeamgo get session logs`

**Help**
```
user@host % veeamgo get session logs --help
Show session log records

Usage:
  veeamgo get session logs [flags]

Flags:
      --id string      Session ID whose logs to fetch
      --status string  Filter log records by status
  -h, --help           help for logs
```

**Success**
```
user@host % veeamgo get session logs --id f848e90c-7f37-4ff5-9d55-04e33f8a4de3 --status Succeeded
ID  Status     Started                   Updated                   Title                                             Additional Info
10  Succeeded  2023-11-05 06:03:15 (+01:00)  2023-11-05 06:03:15 (+01:00)  Primary bottleneck: Source
9   Succeeded  2023-11-05 06:03:15 (+01:00)  2023-11-05 06:03:15 (+01:00)  Load: Source 86% > Proxy 54% > Network 56% > Target 42%
7   Succeeded  2023-11-05 06:00:24 (+01:00)  2023-11-05 06:03:08 (+01:00)  Processing ubuntu88
5   Succeeded  2023-11-05 06:00:18 (+01:00)  2023-11-05 06:02:18 (+01:00)  Processing winsrv100
```

**Error**
```
user@host % veeamgo get session logs
Error: flag --id is required (provide the session ID to inspect)
```

### `veeamgo describe session`

**Help**
```
user@host % veeamgo describe session --help
Show detailed information about a session

Usage:
  veeamgo describe session [flags]

Flags:
  -h, --help        help for session
      --id string   Session ID to describe

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo describe session --id f848e90c-7f37-4ff5-9d55-04e33f8a4de3
ID:              f848e90c-7f37-4ff5-9d55-04e33f8a4de3
Name:            Backup Configuration Job
Type:            ConfigurationBackup
State:           Stopped
Result:          Success (Success)
Progress:        100%
Created At:      2023-11-08 10:00:12 (+01:00)
Ended At:        2023-11-08 10:00:34 (+01:00)
Job ID:          99d1bf3d-e2e0-4bec-b2b3-820c0b87d212
Parent Session ID:
Resource ID:
Resource Ref:
Platform:        VMware
Initiated By:
```

**Error**
```
user@host % veeamgo describe session
Error: flag --id is required (provide the session ID to describe)
Usage:
  veeamgo describe session [flags]
```

---

## Task Commands

### `veeamgo get task`

**Help**
```
user@host % veeamgo get task --help
List task sessions

Usage:
  veeamgo get task [flags]

Flags:
      --created-before string   Include tasks created on/before RFC3339 timestamp
      --created-since string    Include tasks created on/after RFC3339 timestamp
      --desc                    Sort in descending order (default ascending)
      --ended-before string     Include tasks ended on/before RFC3339 timestamp
      --ended-since string      Include tasks ended on/after RFC3339 timestamp
      --limit int               Maximum number of task sessions to return (default: all)
      --name string             Filter by task name (supports * wildcards)
      --result string           Filter by task result
      --scan-result string      Filter by scan result
      --scan-state string       Filter by scan state
      --scan-type string        Filter by scan type
      --session-id string       Filter by parent session ID
      --session-type string     Filter by parent session type
      --sort string             Sort task sessions by column (e.g. creationTime, endTime)
      --state string            Filter by task state
      --type string             Filter by task type
  -h, --help                    help for task
```

**Success**
```
user@host % veeamgo get task --session-id 54979d08-a3a2-44b7-bbfe-6703585d58a3 --limit 1
Name       Type    Session Type  State    Result   Progress  Created                 Ended                   Session ID                         Task ID
linbase01  Backup  BackupJob     Stopped  Success             2024-11-09 18:00:30     2024-11-09 18:04:09     54979d08-a3a2-44b7-bbfe-6703585d58a3  3d995992-0230-42ef-bb30-ae8da82ea24a
```

**Error**
```
user@host % veeamgo get task --created-since invalid
Error: invalid time "invalid" (use RFC3339)
```

### `veeamgo get task logs`

**Help**
```
user@host % veeamgo get task logs --help
Show task session log records

Usage:
  veeamgo get task logs [flags]

Flags:
      --id string      Task session ID whose logs to fetch
      --status string  Filter log records by status
  -h, --help           help for logs
```

**Success**
```
user@host % veeamgo get task logs --id 3d995992-0230-42ef-bb30-ae8da82ea24a --limit 4
ID      Status     Started                   Updated                   Title                                               Additional Info
20248   Succeeded  2024-11-10 18:00:57 (+01:00)  2024-11-10 18:00:57 (+01:00)  Saving [prgtwesx01-ds01] linbase01/linbase01.vmx
20251   Succeeded  2024-11-10 18:01:21 (+01:00)  2024-11-10 18:01:29 (+01:00)  Hard disk 1 (20 GB)                           785 MB read at 107 MB/s [CBT]
20252   Succeeded  2024-11-10 18:01:45 (+01:00)  2024-11-10 18:01:48 (+01:00)  Removing VM snapshot
20217   Succeeded  2024-11-10 18:00:36 (+01:00)  2024-11-10 18:00:36 (+01:00)  Queued for processing at 11/10/2024 6:00:36 PM
```

**Error**
```
user@host % veeamgo get task logs
Error: flag --id is required (provide the task session ID)
```

### `veeamgo describe task`

**Help**
```
user@host % veeamgo describe task --help
Show detailed information about a task session

Usage:
  veeamgo describe task [flags]

Flags:
  -h, --help        help for task
      --id string   Task session ID to describe

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo describe task --id 3d995992-0230-42ef-bb30-ae8da82ea24a
Task ID:      3d995992-0230-42ef-bb30-ae8da82ea24a
Name:         linbase01
Type:         Backup
Session Type: BackupJob
Session ID:   54979d08-a3a2-44b7-bbfe-6703585d58a3
State:        Stopped
Result:       Success (Success)
Progress:
Created At:   2024-11-09 18:00:30 (+00:00)
Ended At:     2024-11-09 18:04:09 (+00:00)
Payload:
  algorithm: Increment
  creationTime: "2024-11-09T18:00:30.407023"
  endTime: "2024-11-09T18:04:09.960857"
  id: 3d995992-0230-42ef-bb30-ae8da82ea24a
  progress:
    bottleneck: Source
    duration: 00:03:39
    processingRate: 85 MB
    processedSize: 21474836480
    readSize: 891289600
    transferedSize: 354695419
  repositoryId: 88788f9e-d8f5-4eb4-bc4f-9b3f5403bcec
  result:
    isCanceled: false
    message: Success
    result: Success
  restorePointId: e7853593-49ac-49d9-be48-bcf98bf330f7
  restorePointReference: /api/v1/restorePoints/e7853593-49ac-49d9-be48-bcf98bf330f7
  sessionId: 54979d08-a3a2-44b7-bbfe-6703585d58a3
  sessionType: BackupJob
  state: Stopped
  type: Backup
  usn: 129809
```

**Error**
```
user@host % veeamgo describe task
Error: flag --id is required (provide the task session ID to describe)
Usage:
  veeamgo describe task [flags]
```

---

## Backup File & Object Commands

### `veeamgo get backup files`

**Help**
```
user@host % veeamgo get backup files --help
List files contained in a backup

Usage:
  veeamgo get backup files [flags]

Flags:
      --backup string          Backup name (optional if job has a single backup)
      --created-before string  Only include files created on/before RFC3339 timestamp
      --created-since string   Only include files created on/after RFC3339 timestamp
      --desc                   Sort in descending order
      --file string            Filter by backup file name (supports * wildcards)
      --gfs string             Filter by GFS period (e.g. Weekly, Monthly)
      --limit int              Maximum number of files to return
      --name string            Backup job name (required)
      --sort string            Sort by column (e.g. creationTime, backupSize)
  -h, --help                   help for files
```

**Success**
```
user@host % veeamgo get backup files --name "AgentBackupJob" --limit 1
Name                                                              Backup Size  Data Size   Dedup Ratio  Compress Ratio  Created                 GFS
C:\Backup\SRV99_Administrator\linbase02 AgentBackupJob\AgentBackupJob_2024-11-06T003020.vib  496.94 MiB  1.12 GiB    81           53              2024-11-06 00:30:20
```

**Error**
```
user@host % veeamgo get backup files
Error: flag --name is required (specify the backup job name)
```

### `veeamgo get backup objects`

**Help**
```
user@host % veeamgo get backup objects --help
List objects protected by a backup

Usage:
  veeamgo get backup objects [flags]

Flags:
      --backup string   Backup name (optional if job has a single backup)
      --name string     Backup job name (required)
  -h, --help            help for objects
```

**Success**
```
user@host % veeamgo get backup objects --name "AgentBackupJob"
Name      Type  Platform  Restore Points  Last Run Result
winsrv88  VM    VmWare    13              Success
```

**Error**
```
user@host % veeamgo get backup objects
Error: flag --name is required (specify the backup job name)
```

---

## Restore Point Commands

### `veeamgo get restorepoint`

**Help**
```
user@host % veeamgo get restorepoint --help
List restore points for a backup job

Usage:
  veeamgo get restorepoint [flags]

Flags:
      --backup string         Backup name (optional if job has a single backup)
      --created-before string Include restore points created on/before RFC3339 timestamp
      --created-since string  Include restore points created on/after RFC3339 timestamp
      --desc                  Sort in descending order
      --limit int             Maximum number of restore points to return
      --malware string        Filter by malware status
      --name string           Backup job name (required)
      --object string         Filter by backup object ID
      --platform string       Filter by platform name (e.g. VMware)
      --platform-id string    Filter by platform ID
      --restorepoint string   Filter by restore point name (supports * wildcards)
      --sort string           Sort by column (e.g. creationTime)
  -h, --help                  help for restorepoint
```

**Success**
```
user@host % veeamgo get restorepoint --name "AgentBackupJob" --limit 1
Object Name                Type       Platform        Created                 Job Name         Malware Status
linbase02 AgentBackupJob   Increment  LinuxPhysical   2024-11-06 00:30:20 (+01:00)  AgentBackupJob   Clean
```

**Error**
```
user@host % veeamgo get restorepoint
Error: --name is required
```

---

## Replica Commands

### `veeamgo get replica`

**Help**
```
user@host % veeamgo get replica --help
List replica restore points for a replication job

Usage:
  veeamgo get replica [flags]

Flags:
      --created-before string  Include restore points created on/before RFC3339 timestamp
      --created-since string   Include restore points created on/after RFC3339 timestamp
      --desc                   Sort in descending order
      --limit int              Maximum number of restore points to return
      --malware string         Filter by malware status
      --name string            Replication job name (required)
      --platform string        Filter by platform name (e.g. VMware)
      --platform-id string     Filter by platform ID
      --replica string         Replica name (optional if job has a single replica)
      --replica-point string   Filter by replica restore point name (supports * wildcards)
      --sort string            Sort by column (e.g. creationTime)
  -h, --help                   help for replica
```

**Success**
```
user@host % veeamgo get replica --name "Production Replica" --limit 1
VM Name  State  Platform  Created                 Job Name            Malware Status
ubuntu88 Ready  VMware    2023-10-26 15:39:53 (+02:00)  Production Replica  Clean
```

**Error**
```
user@host % veeamgo get replica
Error: required flag(s) "name" not set
```

---

## Proxy Commands

### `veeamgo get proxy`

**Help**
```
user@host % veeamgo get proxy --help
List backup proxies

Usage:
  veeamgo get proxy [flags]

Aliases:
  proxy, get

Flags:
      --desc             Sort results in descending order
  -h, --help             help for proxy
      --host-id string   Filter by host identifier
      --limit int        Maximum number of records to return (default: all)
      --name string      Filter by proxy name (supports * wildcards)
      --sort string      Sort results by column (e.g. name)
      --type string      Filter by proxy type (e.g. ViProxy, HvProxy)

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo get proxy --limit 2
Name                Type           Host          Max Tasks  Online  Disabled  Out Of Date
Backup Proxy        General Proxy  This server   2          Yes     No        No
VMware Backup Proxy VMware Proxy   This server   2          Yes     No        No
```

**Error**
```
user@host % veeamgo get proxy --unknown
Error: unknown flag: --unknown
Usage:
  veeamgo get proxy [flags]
```

### `veeamgo describe proxy`

**Help**
```
user@host % veeamgo describe proxy --help
Describe a backup proxy

Usage:
  veeamgo describe proxy [flags]

Aliases:
  proxy, show

Flags:
  -h, --help          help for proxy
      --name string   Proxy name to describe
      --type string   Filter by proxy type when resolving name

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo describe proxy --name "VMware vSphere Backup Proxy"
ID:                    18b661c1-d9dc-4233-90a0-7e7b10dc2d09
Name:                  VMware vSphere Backup Proxy
Type:                  VMware Proxy
API Type:              ViProxy
Description:           Created by TECH\sheila.d.cory
Host ID:               6745a759-2205-4cd2-b172-8ec8f7e60ef8
Host:                  This server
Max Tasks:             2
Transport Mode:        network
Failover To Network:   No
Host To Proxy Encryption: No
Auto Select Datastores: Yes
Connected Datastores:
```

**Error**
```
user@host % veeamgo describe proxy
Error: flag --name is required (provide the proxy name)
Usage:
  veeamgo describe proxy [flags]
```

---

## Configuration Backup Commands

### `veeamgo describe configurationbackup`

**Help**
```
user@host % veeamgo describe configurationbackup --help
Describe configuration backup settings

Usage:
  veeamgo describe configurationbackup [flags]

Flags:
  -h, --help   help for configurationbackup

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo describe configurationbackup
Enabled:              Yes
Repository ID:        d4b5e196-f3ad-474c-99bc-dfef051dae07
Restore Points To Keep: 10
Encryption Enabled:    Yes
Encryption Password ID: ebf6c20f-7126-4186-a1b8-24e6c541161c
Last Successful Run:   2024-01-21 10:00:35 (+01:00)
Last Session ID:       a8f1b5ba-a6dc-416d-9b17-95e92c0d7e76
```

**Error**
```
user@host % veeamgo describe configurationbackup --unknown
Error: unknown flag: --unknown
Usage:
  veeamgo describe configurationbackup [flags]
```

---

## Traffic Rule Commands

### `veeamgo get trafficrule`

**Help**
```
user@host % veeamgo get trafficrule --help
List traffic rules and preferred networks

Usage:
  veeamgo get trafficrule [flags]

Flags:
  -h, --help   help for trafficrule

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo get trafficrule
Name      Source Range  Target Range  Encryption  Throttling  Limit Value  Limit Unit   Time Window
Internet  Any           Internet      Yes         No          2            MbitPerSpec  Yes
```

**Error**
```
user@host % veeamgo get trafficrule --unknown
Error: unknown flag: --unknown
Usage:
  veeamgo get trafficrule [flags]
```

---

## Exclusion VM Commands

### `veeamgo get exclusionvm`

**Help**
```
user@host % veeamgo get exclusionvm --help
List globally excluded VMs

Usage:
  veeamgo get exclusionvm [flags]

Flags:
      --desc          Sort results in descending order
  -h, --help          help for exclusionvm
      --limit int     Maximum number of records to return (default: all)
      --sort string   Sort results by column (e.g. id)

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo get exclusionvm --limit 2
id                                   name     platform  note
497f6eca-6276-4993-bfeb-53cbbbba6f08 dlsql02  VSphere   Exclude due to maintenance
06ad25e5-9284-4e4a-bdb8-44fd0cd3df45 dlsql01  VSphere   Exclude due to maintenance
```

**Error**
```
user@host % veeamgo get exclusionvm --unknown
Error: unknown flag: --unknown
Usage:
  veeamgo get exclusionvm [flags]
```

### `veeamgo describe exclusionvm`

**Help**
```
user@host % veeamgo describe exclusionvm --help
Describe a VM exclusion entry

Usage:
  veeamgo describe exclusionvm [flags]

Aliases:
  exclusionvm, show

Flags:
  -h, --help   help for exclusionvm

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo describe exclusionvm 497f6eca-6276-4993-bfeb-53cbbbba6f08
id:           497f6eca-6276-4993-bfeb-53cbbbba6f08
name:         dlsql02
platform:     VSphere
inventoryId:  vm-52009
note:         Exclude due to maintenance
```

**Error**
```
user@host % veeamgo describe exclusionvm
Error: accepts 1 arg(s), received 0
Usage:
  veeamgo describe exclusionvm [flags]
```

---

## Inventory Commands

### `veeamgo get inventory`

**Help**
```
user@host % veeamgo get inventory --help
Explore inventory resources

Usage:
  veeamgo get inventory [flags]
  veeamgo get inventory [command]

Available Commands:
  protectiongroup List protection groups
  unstructured    List unstructured data servers
  virtualinfra    List virtual infrastructure servers

Flags:
  -h, --help   help for inventory

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use

Use "veeamgo get inventory [command] --help" for more information about a command.
```

**Error**
```
user@host % veeamgo get inventory --unknown
Error: unknown flag: --unknown
Usage:
  veeamgo get inventory [flags]
  veeamgo get inventory [command]
```

### `veeamgo get inventory virtualinfra`

**Help**
```
user@host % veeamgo get inventory virtualinfra --help
List virtual infrastructure servers

Usage:
  veeamgo get inventory virtualinfra [flags]
  veeamgo get inventory virtualinfra [command]

Available Commands:
  objects     List inventory objects for a virtual infrastructure server

Flags:
      --desc               Sort in descending order
  -h, --help               help for virtualinfra
      --host string        Filter by host name
      --limit int          Maximum records to return (default 200) (default 200)
      --name string        Filter by server name (case insensitive substring)
      --platform strings   Filter by platform (vsphere|hyperv|clouddirector)
      --skip int           Number of records to skip
      --sort string        Sort by field (e.g. name)
      --type strings       Filter by inventory type (e.g. VCenterServer, Scvmm)

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use

Use "veeamgo get inventory virtualinfra [command] --help" for more information about a command.
```

**Success**
```
user@host % veeamgo get inventory virtualinfra --limit 2
Name                          Type               Host                          Size
vcenter01.tech.local          vCenterServer      vcenter01.tech.local          N/A
prgclouddirector02.tech02.local  CloudDirectorServer  prgclouddirector02.tech02.local  N/A
```

**Error**
```
user@host % veeamgo get inventory virtualinfra --unknown
Error: unknown flag: --unknown
Usage:
  veeamgo get inventory virtualinfra [flags]
```

### `veeamgo get inventory virtualinfra objects`

**Help**
```
user@host % veeamgo get inventory virtualinfra objects --help
List inventory objects for a virtual infrastructure server

Usage:
  veeamgo get inventory virtualinfra objects [flags]

Flags:
      --desc                  Sort child objects descending
  -h, --help                  help for objects
      --hierarchy string      Hierarchy type (e.g. HostsAndClusters, VmsAndTemplates)
      --limit int             Maximum child objects to return (default 200) (default 200)
      --name string           Server or host name to browse
      --object-name string    Filter objects by name (case insensitive substring)
      --object-type strings   Filter objects by type (e.g. Datacenter, VirtualMachine)
      --skip int              Number of child objects to skip
      --sort string           Sort child objects by field (e.g. name)

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo get inventory virtualinfra objects --name vcenter01.tech.local --limit 3
Name              Type            Host                    Object ID        Size
Prague            Datacenter      vcenter01.tech.local    datacenter-42371 N/A
Templates         Folder          vcenter01.tech.local    group-h42373     N/A
esx03.tech.local  Host            vcenter01.tech.local    host-42428       N/A
```

**Error**
```
user@host % veeamgo get inventory virtualinfra objects --limit 1
Error: flag --name is required (provide the server or host name to inspect)
Usage:
  veeamgo get inventory virtualinfra objects [flags]
```

### `veeamgo get inventory unstructured`

**Help**
```
user@host % veeamgo get inventory unstructured --help
List unstructured data servers

Usage:
  veeamgo get inventory unstructured [flags]

Flags:
      --contains string   Filter servers whose name contains the provided value (case insensitive)
      --desc              Sort in descending order
  -h, --help              help for unstructured
      --limit int         Maximum records to return (default 200) (default 200)
      --name string       Filter servers by name pattern (supports * wildcards)
      --skip int          Number of records to skip
      --sort string       Sort by column (Name or Description)
      --type strings      Filter servers by type (FileServer, SMBShare, NFSShare, NASFiler, S3Compatible, AmazonS3, AzureBlob)

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo get inventory unstructured --limit 2
Name                  Type       Location                       ID
prg-nas01             NASFiler   \\prg-nas01\archives           b7d4d7d1-9da1-49bf-8c0b-8a7f6a8d9a54
SMB Share (Finance)   SMBShare   \\fileserver01\finance         92bd275f-2c6a-42e9-9d6d-fcb10f4a7c21
```

**Error**
```
user@host % veeamgo get inventory unstructured --unknown
Error: unknown flag: --unknown
Usage:
  veeamgo get inventory unstructured [flags]
```

### `veeamgo get inventory protectiongroup`

**Help**

```
user@host % veeamgo get inventory protectiongroup --help
List protection groups

Usage:
  veeamgo get inventory protectiongroup [flags]
  veeamgo get inventory protectiongroup [command]

Available Commands:
  agents      List discovered agents for a protection group

Flags:
      --desc          Sort in descending order
  -h, --help          help for protectiongroup
      --limit int     Maximum records to return (default 200) (default 200)
      --name string   Filter protection groups by name (supports * wildcards)
      --sort string   Sort by column (e.g. name)
      --type string   Filter protection groups by type

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use

Use "veeamgo get inventory protectiongroup [command] --help" for more information about a command.
```

**Success**
```
user@host % veeamgo get inventory protectiongroup --limit 2
Name                     Type                 Description
Manually Added           ManuallyAdded        Contains manually added computers
Microsoft SQL Server PG  IndividualComputers  Created by .\Administrator at 4/8/2025 5:51 PM.
```

**Error**
```
user@host % veeamgo get inventory protectiongroup --unknown
Error: unknown flag: --unknown
Usage:
  veeamgo get inventory protectiongroup [flags]
```

### `veeamgo get inventory protectiongroup agents`

**Help**
```
user@host % veeamgo get inventory protectiongroup agents --help
List discovered agents for a protection group

Usage:
  veeamgo get inventory protectiongroup agents [flags]

Flags:
  -h, --help          help for agents
      --limit int     Maximum agents to return (default 200) (default 200)
      --name string   Protection group name (required)

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo get inventory protectiongroup agents --name "Microsoft SQL Server PG" --limit 1
Protection Group          Server  IP Address      Last Seen                 Backup Agent  Application Plugin  Operating System
Microsoft SQL Server PG   dlsql03 172.24.29.220   2025-04-22 19:01:26 (+00:00)  Yes           Yes                WindowsServer2022
```

**Error**
```
user@host % veeamgo get inventory protectiongroup agents --limit 1
Error: flag --name is required (provide the protection group name)
Usage:
  veeamgo get inventory protectiongroup agents [flags]
```

### `veeamgo describe inventory virtualinfra`

**Help**
```
user@host % veeamgo describe inventory virtualinfra --help
Describe a virtual infrastructure server

Usage:
  veeamgo describe inventory virtualinfra [flags]

Flags:
      --desc          Sort records descending
  -h, --help          help for virtualinfra
      --limit int     Maximum records to inspect (default 200) (default 200)
      --name string   Server or host name to describe
      --skip int      Number of records to skip
      --sort string   Sort records by field (e.g. name)

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo describe inventory virtualinfra --name vcenter01.tech.local
Name:      vcenter01.tech.local
Type:      vCenterServer
Host:      vcenter01.tech.local
Object ID: b9faa59e-c6cd-44ed-8496-1b429a8f9aca
URN:       vc:vcenter01.tech.local
Platform:  VMware
Size:      N/A
```

**Error**
```
user@host % veeamgo describe inventory virtualinfra
Error: flag --name is required (provide the server or host name to describe)
Usage:
  veeamgo describe inventory virtualinfra [flags]
```

### `veeamgo describe inventory unstructured`

**Help**
```
user@host % veeamgo describe inventory unstructured --help
Describe an unstructured data server

Usage:
  veeamgo describe inventory unstructured [flags]

Flags:
  -h, --help          help for unstructured
      --id string     Server ID to describe
      --limit int     Maximum records to inspect when resolving by name (default 200) (default 200)
      --name string   Server name to describe

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo describe inventory unstructured --name "SMB Share (Finance)"
Name:                SMB Share (Finance)
Type:                SMBShare
Location:            \\fileserver01\finance
ID:                  92bd275f-2c6a-42e9-9d6d-fcb10f4a7c21
Credentials Required: Yes
Credentials ID:       34e49092-21fc-4a09-b9e6-8009f8d77d94
Cache Repository:     88788f9e-d8f5-4eb4-bc4f-9b3f5403bcec
```

**Error**
```
user@host % veeamgo describe inventory unstructured
Error: flag --name or --id is required (provide the server name or ID to describe)
Usage:
  veeamgo describe inventory unstructured [flags]
```

### `veeamgo describe inventory protectiongroup agent`

**Help**
```
user@host % veeamgo describe inventory protectiongroup agent --help
Describe a protection group agent

Usage:
  veeamgo describe inventory protectiongroup agent [flags]

Flags:
      --group string   Restrict search to a specific protection group
  -h, --help           help for agent
      --name string    Agent (entity) name to describe

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo describe inventory protectiongroup agent --name dlsql03 --group "Microsoft SQL Server PG"
Protection Group:          Microsoft SQL Server PG
Name:                      dlsql03
Type:                      Computer
State:                     Online
Agent Status:              Installed
Agent Version:             13.0.0.632
Driver Status:             Installed
Driver Version:            Veeam Volume CT Driver (13.0.0.12)
Reboot Required:           No
IP Addresses:              172.24.29.220
Last Seen:                 2025-04-23 08:18:33 (+00:00)
Operating System:          WindowsServer2022
Operating System Version:  Microsoft Windows Server 2022 (21H2, 64-bit)
Platform:                  X64
```

**Error**
```
user@host % veeamgo describe inventory protectiongroup agent --group "Microsoft SQL Server PG"
Error: flag --name is required (provide the agent name)
Usage:
  veeamgo describe inventory protectiongroup agent [flags]
```

---

## License Commands

### `veeamgo get license`

**Help**
```
user@host % veeamgo get license --help
License usage and allocations

Usage:
  veeamgo get license [flags]
  veeamgo get license [command]

Available Commands:
  capacity    List capacity license workloads
  instances   List instance-based workloads
  sockets     List socket-based workloads
  summary     Show license summary

Flags:
  -h, --help   help for license

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use

Use "veeamgo get license [command] --help" for more information about a command.
```

**Success**
```
user@host % veeamgo get license
Licensed To                     Edition        Type          Status  Expires At                    Support Expires At
Veeam Software Group GmbH       EnterprisePlus Subscription  Valid   2026-12-12 00:00:00 (+00:00)
```

**Error**
```
user@host % veeamgo get license --unknown
Error: unknown flag: --unknown
Usage:
  veeamgo get license [flags]
  veeamgo get license [command]
```

### `veeamgo get license sockets`

**Help**
```
user@host % veeamgo get license sockets --help
List socket-based workloads

Usage:
  veeamgo get license sockets [flags]

Aliases:
  sockets, socket

Flags:
      --cores int          Filter by core count
      --desc               Sort results in descending order
  -h, --help               help for sockets
      --host-id string     Filter by proxy host id
      --host-name string   Filter by proxy host name
      --limit int          Maximum number of records to return (default: all)
      --name string        Filter by workload name (supports * wildcards)
      --sockets int        Filter by socket count
      --sort string        Sort results by column (e.g. name)
      --type string        Filter by workload type

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo get license sockets --limit 1
Workload                 Host                    Type      Sockets  Cores  Host ID
prgtwesx01.tech.local    prgtwesx01.tech.local   vSphere   2        16     a6ca6129-5c58-47e3-b35c-ff1187d88ee4
```

**Error**
```
user@host % veeamgo get license sockets --unknown
Error: unknown flag: --unknown
Usage:
  veeamgo get license sockets [flags]
```

### `veeamgo get license instances`

**Help**
```
user@host % veeamgo get license instances --help
List instance-based workloads

Usage:
  veeamgo get license instances [flags]

Aliases:
  instances, instance

Flags:
      --desc                 Sort results in descending order
  -h, --help                 help for instances
      --host-name string     Filter by host name
      --instance-id string   Filter by instance id
      --limit int            Maximum number of records to return (default: all)
      --name string          Filter by workload name (supports * wildcards)
      --sort string          Sort results by column (e.g. usedInstancesNumber)
      --type string          Filter by workload type
      --used float           Filter by consumed instances

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo get license instances --limit 3
Workload  Host                     Type         Platform  Used Instances  Revocable
rhel01    vcenter01.tech.local     VM           VMware    1.00            No
mongodb01 productionReplicaSet     Application  MongoDB   1.00            No
mongodb02 productionReplicaSet     Application  MongoDB   1.00            No
```

**Error**
```
user@host % veeamgo get license instances --unknown
Error: unknown flag: --unknown
Usage:
  veeamgo get license instances [flags]
```

### `veeamgo get license capacity`

**Help**
```
user@host % veeamgo get license capacity --help
List capacity license workloads

Usage:
  veeamgo get license capacity [flags]

Aliases:
  capacity, capacity-workloads

Flags:
  -h, --help   help for capacity

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo get license capacity
Workload                          Type       Used (TB)  Instance ID
winsrv88:/C:\Shares\nfs_share     FileShare  0.00       0405a174-dc1a-473a-a2cf-b5b1c8b1b620
```

**Error**
```
user@host % veeamgo get license capacity --unknown
Error: unknown flag: --unknown
Usage:
  veeamgo get license capacity [flags]
```

---

## Security Analyzer Commands

### `veeamgo start securityanalyzer`

**Help**
```
user@host % veeamgo start securityanalyzer --help
Start Security & Compliance Analyzer

Usage:
  veeamgo start securityanalyzer [flags]

Flags:
      --wait   Wait for the analyzer session to finish
      --yes    Confirm without prompting (required to execute)
  -h, --help   help for securityanalyzer

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo start securityanalyzer --yes
Started Security & Compliance Analyzer. Session 19b1f78d-0c81-4e42-8d26-51c8923b8031 (Running).
```

**Success (`--wait`)**
```
user@host % veeamgo start securityanalyzer --wait --yes
Security & Compliance Analyzer finished. Session f8c7af6e-f343-4c0a-b2a3-1a542a9974fb (Stopped).

Best Practice                                     Status    Note
Enable immutable backups                          Warning   Configure immutable storage for primary backups.
Require multifactor authentication for console    Passed
Rotate credentials for service accounts           Failed    Rotate the service account password used by backup jobs.
```

**Error**
```
user@host % veeamgo start securityanalyzer
Start Security & Compliance Analyzer [y/N]: n
Cancelled start request.
```

### `veeamgo start configurationbackup`

**Help**
```
user@host % veeamgo start configurationbackup --help
Start an on-demand configuration backup

Usage:
  veeamgo start configurationbackup [flags]

Flags:
      --wait   Wait for the backup session to finish
      --yes    Confirm without prompting (required to execute)
  -h, --help   help for configurationbackup

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo start configurationbackup --yes
Started configuration backup. Session 4c9b7e43-2c60-41b2-92f4-7f1d3d572011 (Running).
```

**Error**
```
user@host % veeamgo start configurationbackup
Start configuration backup [y/N]: n
Cancelled start request.
```

### `veeamgo get securityanalyzer results`

**Help**
```
user@host % veeamgo get securityanalyzer results --help
List analyzer compliance results

Usage:
  veeamgo get securityanalyzer results [flags]

Flags:
  -h, --help   help for results

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo get securityanalyzer results
Best Practice  Status   Note
Enable MFA     Warning  Pending
```

**Error**
```
user@host % veeamgo get securityanalyzer results --unknown
Error: unknown flag: --unknown
Usage:
  veeamgo get securityanalyzer results [flags]
```

### `veeamgo describe securityanalyzer schedule`

**Help**
```
user@host % veeamgo describe securityanalyzer schedule --help
Show analyzer schedule configuration

Usage:
  veeamgo describe securityanalyzer schedule [flags]

Flags:
  -h, --help   help for schedule

Global Flags:
      --config string    Path to configuration file
      --output string    Output format (table|json) (default "table")
      --profile string   Profile name to use
```

**Success**
```
user@host % veeamgo describe securityanalyzer schedule
Daily Scan Enabled: Yes
Daily Scan Time:    07:30
Send Results:       Yes
Recipients:         soc@veeamgo.test; security@veeamgo.test
Notification Type:  Custom
Custom Subject:     [Analyzer] Results for %DATE%
Notify On Success:  No
Notify On Warning:  Yes
Notify On Error:    Yes
```

**Error**
```
user@host % veeamgo describe securityanalyzer schedule --unknown
Error: unknown flag: --unknown
Usage:
  veeamgo describe securityanalyzer schedule [flags]
```

---

> **Note:** Rescan and other mutating verbs should follow the same documentation pattern—capture help, a representative success output, and a common error scenario when you extend this reference.
