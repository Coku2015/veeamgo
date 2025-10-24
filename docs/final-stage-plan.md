## Final Stage Plan – Verified Against OpenAPI Spec

The table below lists remaining CLI features to implement. Each item references the exact REST API operation ID and path from `docs/swagger/openapi.json` so we only target capabilities explicitly exposed by the API.

### 1. Managed Server Maintenance
| CLI Goal | Operation ID | Method & Path |
|----------|--------------|---------------|
| Remove managed server by ID | `DeleteManagedServer` | `DELETE /api/v1/backupInfrastructure/managedServers/{id}` |
| Update managed server metadata (e.g., description) | `UpdateManagedServer` | `PUT /api/v1/backupInfrastructure/managedServers/{id}` |
| Refresh optional components after host changes | `UpdateServerComponents` | `POST /api/v1/backupInfrastructure/managedServers/updateComponents` |
| Update single-use credentials on an existing host | `UpdateSingleUseCredentials` | `POST /api/v1/backupInfrastructure/managedServers/{id}/updateSingleUseCredentials` |
| Inspect & update CBT volumes | `GetManagedServerVolume` / `UpdateVolume` | `GET /api/v1/backupInfrastructure/managedServers/{id}/volumes`<br>`PUT /api/v1/backupInfrastructure/managedServers/{id}/volumes` |

### 2. Mount Server Management
| CLI Goal | Operation ID | Method & Path |
|----------|--------------|---------------|
| List all mount servers | `GetAllMountServers` | `GET /api/v1/backupInfrastructure/mountServers` |
| Create a mount server role on a host | `SetupMountServer` | `POST /api/v1/backupInfrastructure/mountServers` |
| View default mount server | `GetDefaultMountServer` | `GET /api/v1/backupInfrastructure/mountServers/default` |
| Set default mount server | `SetAsDefaultMountServer` | `POST /api/v1/backupInfrastructure/mountServers/{id}/default` |
| Describe/update specific mount server | `GetMountServer` / `UpdateMountServer` | `GET /api/v1/backupInfrastructure/mountServers/{id}`<br>`PUT /api/v1/backupInfrastructure/mountServers/{id}` |

### 3. Scale-out Backup Repositories (SOBR)
| CLI Goal | Operation ID | Method & Path |
|----------|--------------|---------------|
| List SOBRs | `GetAllScaleOutRepositories` | `GET /api/v1/backupInfrastructure/scaleOutRepositories` |
| Describe SOBR | `GetScaleOutRepository` | `GET /api/v1/backupInfrastructure/scaleOutRepositories/{id}` |
| Create SOBR | `CreateScaleOutRepository` | `POST /api/v1/backupInfrastructure/scaleOutRepositories` |
| Edit SOBR | `UpdateScaleOutRepository` | `PUT /api/v1/backupInfrastructure/scaleOutRepositories/{id}` |
| Delete SOBR | `DeleteScaleOutRepository` | `DELETE /api/v1/backupInfrastructure/scaleOutRepositories/{id}` |
| Toggle maintenance mode for an extent | `EnableScaleOutExtentMaintenanceMode` / `DisableScaleOutExtentMaintenanceMode` | `POST /api/v1/backupInfrastructure/scaleOutRepositories/{id}/enableMaintenanceMode`<br>`POST /api/v1/backupInfrastructure/scaleOutRepositories/{id}/disableMaintenanceMode` |
| Toggle sealed mode for an extent | `EnableScaleOutExtentSealedMode` / `DisableScaleOutExtentSealedMode` | `POST /api/v1/backupInfrastructure/scaleOutRepositories/{id}/enableSealedMode`<br>`POST /api/v1/backupInfrastructure/scaleOutRepositories/{id}/disableSealedMode` |

### 4. WAN Accelerators (Read-only coverage)
Only two operations exist; we can surface them via `veeamgo get/describe wanaccelerator`:
| Operation ID | Method & Path |
|--------------|---------------|
| `GetAllWANAccelerators` | `GET /api/v1/backupInfrastructure/wanAccelerators` |
| `GetWANAccelerator` | `GET /api/v1/backupInfrastructure/wanAccelerators/{id}` |

### 5. Cloud Credentials
| CLI Goal | Operation ID | Method & Path |
|----------|--------------|---------------|
| List credentials | `GetAllCloudCreds` | `GET /api/v1/cloudCredentials` |
| Create credential record | `CreateCloudCreds` | `POST /api/v1/cloudCredentials` |
| Update credential record | `UpdateCloudCreds` | `PUT /api/v1/cloudCredentials/{id}` |
| Delete credential record | `DeleteCloudCreds` | `DELETE /api/v1/cloudCredentials/{id}` |
| Rotate keys / certificates | `ChangeCloudCredsSecretKey`, `ChangeCloudCertificate`, `ChangeAccount` | `POST /api/v1/cloudCredentials/{id}/changeSecretKey`<br>`POST /api/v1/cloudCredentials/{id}/changeCertificate`<br>`POST /api/v1/cloudCredentials/{id}/changeAccount` |
| Manage helper appliances | `GetAllCredsHelperAppliances`, `CreateCloudCredsHelperAppliance`, `GetCloudCredsHelperAppliance`, `DeleteCloudCredsHelperApplianceAsync` | `GET /api/v1/cloudCredentials/{id}/helperAppliances`<br>`POST /api/v1/cloudCredentials/{id}/helperAppliances`<br>`GET /api/v1/cloudCredentials/{id}/helperAppliances/{applianceId}`<br>`DELETE /api/v1/cloudCredentials/{id}/helperAppliances/{applianceId}` |
| OAuth device code flow helpers | `RequestAppRegistrationByDeviceCode`, `FinishAppRegistrationByDeviceCode`, `RequestAuthentication` | `POST /api/v1/cloudCredentials/appRegistration`<br>`POST /api/v1/cloudCredentials/appRegistration/{verificationCode}`<br>`POST /api/v1/cloudCredentials/authenticate` |

### 6. Object Storage Repositories – CLI UX Enhancements
(Implemented via `CreateRepository` / `UpdateRepository`; no new endpoints required)
- Add provider-specific flag shortcuts (e.g., `--bucket-name`, `--endpoint`) on top of the existing `--spec` workflow.
- Expose proxy appliance settings (`proxyAppliance` section within object repository specs) via flags.

### 7. Restore & Mount (Selected High-value Operations)
| CLI Goal | Operation ID | Method & Path |
|----------|--------------|---------------|
| Publish backup content | `PublishBackupContent` | `POST /api/v1/restoreSessions/contentPublishing`
| Unpublish backup content | `UnpublishBackupContent` | `DELETE /api/v1/restoreSessions/contentPublishing/{id}`
| Mount backup (VM/file-level) | `MountBackup` | `POST /api/v1/restoreSessions/mounts`
| Unmount backup | `UnmountBackup` | `DELETE /api/v1/restoreSessions/mounts/{id}`
| Browse FLR sessions (read-only UI helper) | `GetAllFlrBrowseSessions` / `GetFlrBrowseSession` | `GET /api/v1/restoreSessions/flr/browseSessions`<br>`GET /api/v1/restoreSessions/flr/browseSessions/{id}` |

### 8. Diagnostics & Security (Optional CLI coverage)
| Operation ID | Method & Path | Suggested CLI |
|--------------|---------------|----------------|
| `ViewSuspiciousActivityEvents` | `GET /api/v1/security/suspiciousActivities` | `veeamgo get suspicious-activity` |
| `GetAllAuthorizationEvents` | `GET /api/v1/security/authorizationEvents` | `veeamgo get authorization-event` |
| `ExportSupportLogs` / `DownloadSupportLogBundle` | `POST /api/v1/diagnostics/supportLogs`<br>`GET /api/v1/diagnostics/supportLogs/{id}` | `veeamgo start supportlogs`, `veeamgo get supportlog` |

### 9. Testing & Documentation Follow-up
- Add client unit tests for new endpoints (managed server update/delete, SOBR actions, mount server configuration, cloud credential helpers).
- Expand CLI docs (`docs/ui-design-reference.md`, `docs/user-guide.md`) after implementing each new verb.
- Extend smoke script(s) to cover new commands where practical.

This plan is limited to endpoints explicitly published in the OpenAPI spec and can be updated as additional APIs become available.
