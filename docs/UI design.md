(base) lei@Lei-MBP-M2 veeamgo % ./bin/veeamgo server get info
Name:              vbrsav13.backupnext.home
Platform:          Linux
VBR uuid:          99ab520d-aa2c-4189-b908-61557da58330
Build Version:     13.0.0.4967
Database:          PostgreSQL
Database Version:  PostgreSQL 17.6 on x86_64-pc-linux-gnu, compiled by gcc (GCC) 11.5.0 20240719 (Red Hat 11.5.0-5), 64-bit
Database Edition:  
Patches:           

(base) lei@Lei-MBP-M2 veeamgo % ./bin/veeamgo server get time
Server Time:  2025-10-11 13:09:10.9694282 +0800 CST
Timezone:     (UTC+08:00) China Standard Time (Shanghai)

(base) lei@Lei-MBP-M2 veeamgo % ./bin/veeamgo repository get --limit 2
Name                       Type           Host                      Path                                  Capacity(GB)  Free(GB)  Used(GB)  Online  Out of Date  Description                                Id
Default Backup Repository  LinuxHardened  vbrsav13.backupnext.home  /var/lib/veeam/backup                 260.9         259       0         Yes     No           Created by Veeam Backup                    88788f9e-d8f5-4eb4-bc4f-9b3f5403bcec
Synology NFS               Nfs            Gateway (auto)            nfs://10.10.1.45:/volume1/VBRBackups  50042.3       8878.7    6878.9    Yes     No           Created by .\veeamadmin at 2025/9/7 8:43.  526cc0d7-5a43-4f41-b491-1efaa151f492

(base) lei@Lei-MBP-M2 veeamgo % ./bin/veeamgo repository describe "Default Backup Repository"
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


v100lab@v100lab:~$ ./veeamgo job get
Name          Type           Workload  Status   Last Result  Last Run              Next Run              Repository  Objects  High Priority  Last Session                          Progress
Backup Job 1  VSphereBackup  vm        stopped  Success      2025-10-10T14:00:12Z  2025-10-11T14:00:00Z  VHR-01      2        No             199e3af7-2543-4eb8-b4f5-ae0ab615c386  100%


v100lab@v100lab:~$ ./veeamgo job describe "Backup Job 1"
(base) lei@Lei-MBP-M2 veeamgo % ./bin/veeamgo job describe "Backup Job 1"
Name:             Backup Job 1
Type:             VSphereBackup
Workload:         vm
Description:      Created by .\veeamadmin at 9/16/2025 12:57 PM.
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


