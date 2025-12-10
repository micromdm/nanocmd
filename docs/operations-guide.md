# NanoCMD Operations Guide

This is a brief overview of the various flags, APIs, and other topics related to the operation of the NanoCMD server.

## NanoCMD server

### Command line flags

Command line flags can be specified using command line arguments or environment variables (in NanoCMD versions later than v0.6). Flags take precedence over environment variables, which take precedence over default values. Environment variables are denoted in square brackets below (e.g., [HELLO]), and default values are shown in parentheses (e.g., (default "world")). If an environment variable is currently set then the help output will add "is set" as an indicator.

#### -h, -help

Built-in flag that prints all other available flags, environment variables, and defaults.

#### -api string

* API key for API endpoints [NANOCMD_API]

API authorization in NanoCMD is simply HTTP Basic authentication using "nanocmd" as the username and this API key as the password.

#### -debug

* log debug messages [NANOCMD_DEBUG]

Enable additional debug logging.

#### -dump-webhook

* dump webhook input [NANOCMD_DUMP_WEBHOOK]

For each incoming webhook response this flag dumps the HTTP body to standard output. For the "mdm.Connect" (command response) webhook event it also decodes and outputs the raw Plist.

#### -enqueue-api string

* MDM server API key [NANOCMD_ENQUEUE_API]

The API key (HTTP Basic authentication password) for the MDM server enqueue endpoint. The HTTP Basic username depends on the MDM mode. By default it is "nanomdm" but if the `-micromdm` flag is enabled then it is "micromdm".

#### -enqueue-url string

* URL of MDM server enqueue endpoint [NANOCMD_ENQUEUE_URL]

URL of the MDM server for enqueuing commands. The enrollmnet ID is added onto this URL as a path element (or multiple, if the MDM server supports it).

#### -listen string

* HTTP listen address [NANOCMD_LISTEN] (default ":9003")

Specifies the listen address (interface & port number) for the server to listen on.

#### -micromdm

* MicroMDM-style command submission [NANOCMD_MICROMDM]

Submit commands for enqueueing in a style that is compatible with MicroMDM (instead of NanoMDM). Specifically this flag limits sending commands to one enrollment ID at a time, uses a POST request, and changes the HTTP Basic username.

#### -push-url string

* URL of MDM server push endpoint [NANOCMD_PUSH_URL]

URL of the MDM server for sending APNs pushes. The enrollment ID is added onto this URL as a path element (or multiple, if the MDM server supports it).

#### -repush-interval uint

* interval for repushes in seconds [NANOCMD_REPUSH_INTERVAL] (default 86400)
  * Default interval is 1 day.

If an enrollment ID has not seen a response to a command after this interval then NanoCMD sends an APNs notification to the device.

#### -step-timeout uint

 * default step timeout in seconds [NANOCMD_STEP_TIMEOUT] (default 259200)
   * Default timeout is 3 days.

If a step is not completed within this time period the step is cancelled and returned to the workflow for any (optional) processing. Note the client may still respond to the commands (they are not de-queued from the MDM server, merely removed from tracking in NanoCMD).

#### -storage & -storage-dsn

* -storage string
  * name of storage backend [NANOCMD_STORAGE] (default "file")
* -storage-dsn string
  * data source name (e.g. connection string or path) [NANOCMD_STORAGE_DSN]
* -storage-options string
  * storage backend options [NANOCMD_STORAGE_OPTIONS]

The `-storage`, `-storage-dsn`, and `-storage-options` flags together configure the storage backend. `-storage` specifies the storage backend type while `-storage-dsn` specifies the Data Source Name (i.e. the database connection string or location). The optional `-storage-options` flag specifies options for the backend if it supports them. The default storage backend is `file` if no other backend is specified.

##### file storage backend

* `-storage file`

Configures the `file` storage backend. Data is stored in filesystem files and directories, requires zero dependencies, and should just work right out of the box. The `-storage-dsn` flag specifies the filesystem directory under which the database is created. If no DSN is provided then a default of `db` is used.

*Example:* `-storage file -storage-dsn /path/to/my/db`

##### inmem storage backend

* `-storage inmem`

Configures the `inmem` storage backend. Data is stored entirely in-memory and is completely volatile — the database will disappear the moment the server process exits. The `-storage-dsn` flag is ignored for this storage backend.

*Example:* `-storage inmem`

##### mysql storage backend

* `-storage mysql`

Configures the MySQL storage backend. The `-storage-dsn` flag should be in the [format the SQL driver expects](https://github.com/go-sql-driver/mysql#dsn-data-source-name). MySQL 8.0.19 or later is required.. Be sure to create the storage tables with the schema definitions:

* Engine [schema.sql](../storage/mysql/schema.sql)
* Profile subsystem [schema.sql](../subsystem/profile/storage/mysql/schema.sql)

**WARNING:** The MySQL backend currently only implements storage for the workflow *engine* and the profile *subsystem*. When running NanoCMD the other *subsystem* storage is completely in-memory as if you supplied `-storage inmem`. The practical effect is that non-profile subsystem storage is volatile and no data will be persisted for them.

*Example:* `-storage mysql -dsn nanocmd:nanocmd/mycmddb`

#### -version

* print version and exit

Print version and exit.

#### -worker-interval uint

* interval for worker in seconds [NANOCMD_WORKER_INTERVAL] (default 300)
  * Default interval is 5 minutes.

NanoCMD spins up a worker that enqueues future steps, re-pushes to devices, and monitors for timed-out steps. The worker will wake up at this internval to process asynchronous duties. Setting this flag to zero will turn off the worker (effectively disabling those features).

### API endpoints

The NanoCMD server is directed via its REST-ish API. A brief overview of the API endpoints is provided here. For detailed API documentation please refer to the [NanoCMD OpenAPI documentation](https://www.jessepeterson.space/swagger/nanocmd.html). The [OpenAPI source YAML](../docs/openapi.yaml) is part of this project as well. Also take a look at the [QuickStart guide](../docs/quickstart.md) for a tutorial on using the APIs.

Most of the API endpoints are protected by HTTP Basic authentication where the password is specified by the `-api` flag (as documented above).

#### Version endpoint

* Endpoint: `GET /version`

Returns a JSON response with the version of the running NanoCMD server.

#### Webhook endpoint

* Endpoint: `POST /webhook`

The webhook endpoint handles MicroMDM-compatible webhook events. These include MDM command and check-in event responses from MDM clients. See the [MicroMDM documentation for more information](https://github.com/micromdm/micromdm/blob/main/docs/user-guide/api-and-webhooks.md).

#### Workflow Start endpoint

* Endpoint: `POST /v1/workflow/{name}/start`
* Path parameters:
  * `name`: workflow name
* Query parameters:
  * `id`: enrollment ID. multiple supported.
  * `context`: workflow-dependent context (start) value

Starts a workflow.

#### Event Subscription endpoints

* Endpoint: `GET /v1/event/{name}`
* Endpoint: `PUT /v1/event/{name}`
* Path parameters:
  * `name`: user-defined event subscription name

Configures Event Subscriptions. Event Subscriptions start workflows for MDM events. In JSON form they look like this:

```json
{
  "event": "Enrollment",
  "workflow": "io.micromdm.wf.example.v1",
  "context": "string",
  "event_context": "string"
}
```

The JSON keys are as follows:

* `event`: the NanoCMD event name. 
  * `Authenticate`: when a device sends an Authenticate MDM check-in message.
  * `TokenUpdate`: when an enrollment sends a TokenUpdate MDM check-in message.
  * `Enrollment`: when an enrollment enrolls; i.e. the first TokenUpdate message.
  * `CheckOut`: when a device sends a CheckOut MDM check-in message.
  * `Idle`: when an enrollment sends an Idle command response.
  * `IdleNotStartedSince`: when an enrollment sends an Idle message and the associated workflow has not been started in the given number of seconds. The seconds are provided in the `event_context` string.
* `workflow`: the name of the workflow.
* `context`: optional context to give to the workflow when it starts.
* `event_context`: optional context to give to the event.

#### FileVault profile template endpoint

* Endpoint: `GET /v1/fvenable/profiletemplate`

Returns the Configuration Profile template for the FileVault enable workflow. Take care to note the `__CERTIFICATE__` string (which is string-replaced with the actual certificate). This profile can be modified and re-uploaded to the profile store with a name of the FileVault enable workflow. The workflow will attempt to pull that profile and fallback to this hard-coded version if it does not exist.

#### Profile endpoints

* Endpoint: `GET /v1/profile/{name}`
* Endpoint: `PUT /v1/profile/{name}`
* Endpoint: `DELETE /v1/profile/{name}`
* Path parameters:
  * `name`: user-defined profile name

Retrieve, store, or delete profiles by name parameter in the path. Upload raw profiles (including signed profiles) using the `PUT` method. Retrive again with `GET` and of course delete with `DELETE`.

#### Profile list endpoint

* Endpoint `GET /v1/profiles`
* Query parameters:
  * `name`: user-defined profile name. optional. multiple supported.

List the profile UUIDs and identifiers mapped by profile name in profile subsystem storage. Supply the name argument for specific profiles to list.

#### Command Plan endpoints

* Endpoint: `GET /v1/cmdplan/{name}`
* Endpoint: `PUT /v1/cmdplan/{name}`
* Path parameters:
  * `name`: user-defined command plan name

Retrieve and store command plans — collections of MDM actions/commands to be sent together (such as upon device enrollment). **See also** the below discussion of the command plan workflow. Command plans take the JSON form of:

```json
{
  "profile_names": [
    "profile1"
  ],
  "manifest_urls": [
    "https://example.com/manifest"
  ],
  "device_configured": true
}
```

The JSON keys are:

* `profile_names`: list of profiles in the profile subsystem storage. will generate an `InstallProfile` MDM command for each listed item.
* `manifest_urls`: list of URLs to [app installation manifests](https://developer.apple.com/documentation/devicemanagement/manifesturl/itemsitem). will generate an `InstallApplication` MDM command for each URL.
* `device_configured`: if the workflow is started from an enroll event and the device is in the await configuration state then setting this `true` will generate a `DeviceConfigured` MDM command. this will bring the device out of the await configuration state.

#### Inventory endpoint

* Endpoint: `GET /v1/inventory`
* Query parameters:
  * `id`: enrollment ID. multiple supported.

Queries the inventory subsystem to retrieve previously saved inventory data. Inventory key-value data is returned in a JSON object (map) for for each `id` parameter specified.

### Engine

As mentioned in the [README](../README.md) the workflow *engine* is the component that does the heavy lifting of abstracting the MDM command sending and response receiving to provide the workflows with a consistent and easy to use API. It acts as the glue between workflows and MDM servers.

There are a few knobs in the server for the engine: namely the flags `-worker-interval`, `-step-timeout`, and `-repush-interval` documented above. Largely, though, the engine is driven by workflows enqueuing steps and the MDM server sending events. That said the main API endpoints for working with the engine are going to be the Workflow Start endpoint and the Event Subscription endpoints — also documented above. These are the ways ways you kick-off workflows in NanoCMD.

## Subsystems

While they are alluded to the APIs above and workflows below it is worth calling out the *subsystems* themselves. Largely they provide storage backing for their domain specific data as well as the raw HTTP API handlers.

### Command Plan subsystem

The command plan subsystem provides storage backends for command plans. This supports the subsystem's HTTP APIs and of course the actual workflow for retrieving the configurations.

### FileVault subsystem

The FileVault storage subsystem supports two main duties. First the keypair generation, storage, and decryption that allows for devices to encrypt FileVault Pre-Shared Keys (PSKs) to the subsystem-provided public keys. Secondly the FileVault subsystem includes an adapter to the *inventory* subsystem for escrowing (storing) and retrieving PSKs. Effectively this means FileVault esrowed PSKs are stored on directly on the enrollment inventory record.

### Profile subsystem

The profile subsystem provides storage backends for user-named Apple Configuration profiles. This supports the subsystem's HTTP APIs and of course the actual workflow for installing and removing profiles. As well the FileVault workflow uses the profile subsystem for storage.

### Inventory subsystem

The inventory subsystem provides storage backends for "inventory" data — that is, metadata about MDM enrollments. This data is largely collected through the inventory workflow but also data is populated from other workflows such as the FileVault PSK mechanism.

## Workflows

Workflows are domain-specific, contained, and encapsulated MDM command sequence senders and processors. For a higher level review of workflows check out the [README](../README.md). For more information about the internals and implementation of workflows please read [the package documentation](../workflow/doc.go).

### Certificate-Profile Workflow

* Workflow name: `io.micromdm.wf.certprof.v1`
* Start value/context: JSON object, see below.

The certificate-profile workflow conditionally installs configuration profiles (ostensibly certificate identity profiles) based on output from the MDM `CertificateList` command. It does this, roughly, by doing these steps:

1. Sending the MDM `CertificateList` command.
1. Receiving the list of certificates and applying the *filter* (below) to find a single matching certificate.
1. If a certificate is found then checking that it against *criteria* (below) to determine if it needs to be replaced (and thus, the profile being installed)
1. If no certificiate is found then the profile proceeds to be installed.
1. If the certificate exists and criteria is not such that it needs to be replaced, then the workflow ends before installing the profile.

#### Context (input parameters)

The workflow's context determines details about how the workflow runs. The four main keys are:

* `profile`: Profile name in profile storage to install (before text replacement).
* `criteria`: Used to determine whether to replace a found certificate.
* `filter`: Used to find the particular certificate in the list of certificates.
* `text_replacements`: Optionally used to perform simple text find-and-replace on the profile before installing it.

Example JSON object:

```json
{
    "criteria": {
        "always_replace": true
    },
    "filter": {
        "cn_prefix": "kai"
    },
    "profile": "scep",
    "text_replacements": {
        "%CN%": "kai",
        "%CHAL%": "secret"
    }
}
```

### Command Plan Workflow

* Workflow name: `io.micromdm.wf.cmdplan.v1`
* Start value/context: string value of command plan name. See also parameter expansion discussion below.
  * Example: `my_cool_cmdplan`

A command plan (or cmdplan) is a named structured list of operations to send to an enrollment. Each item roughtly corresponds to an MDM command (such as for installing profiles or applications). An example might look like:

```json
{
  "profile_names": [
    "test1"
  ],
  "manifest_urls": [],
  "device_configured": true
}
```

In this example this command plan would send one `InstallProfile` command with the contents of the `test1` profile from the profile subsystem as well as try to send a `DeviceConfigured` command (assuming that it's appropriate for the enrollment at the time — i.e. at the initial Setup Assistant for an ADE enrollment). Command plans themselves are managed via NanoCMD's APIs.

#### Parameter expansion

There is a special parameter expansion mode that the command plan workflow supports when being started. Usually you provide the name of the command plan as the initial context/start value. However you can also provide a shell-like variable substituion based on the URL paremeters that the MDM client is using (which is, ultimately, specified in the MDM enrollment profile).

For example, this is the command plan name that be configured:

```sh
cmdplan_${group}
```

Then, if a client has this in their MDM enrollment profile:

```xml
<key>CheckInURL</key>
<string>https://mdm.example.com/mdm?group=staff</string>
```

Then the command plan workflow will replace the `${group}` variable with the name of the parameter (in this example, `staff`) before using that as the command plan name. I.e. it will use `cmdplan_staff` to lookup the and use the command plan name.

In this way you can have per-device or per-workgroup command plans at enrollment time. This will only work when triggering this workflow from an MDM event (like Enrollment). You'll need to specify the full command plan name when ad-hoc executing a command plan.

Finally this parameter expansion provides a fallback mechanism as well. Follow the parameter name with a colon (`:`) to specify a fallback. For example if `cmdplan_${group:fallback}` is specified and group is *not* in the MDM URL parameters then the expansion would be `cmdplan_fallback`.

#### As compared to the Profile Workflow

*Question:* Both the Command Plan workflow and the Profile Workflow (below) support installing profiles. Which should I use?

In general the Command Plan workflow is meant for installing *sequences* of MDM commands. In particular the there are some Configuration Profile payloads and MDM commands that must be sent while an enrollment (device) is in the Await Configuration state. The Command Plan workflow is well suited for this and in general is probably more suited for ad-hoc or event-driven invocation/starting.

The Profile workflow, on the other hand, is meant more for managing state and if your goal is *continually* make sure profiles are consistent on devices then the Profile workflow is probably a better choice.

### FileVault Enable Workflow

* Workflow name: `io.micromdm.wf.fvenable.v1`
* Start value/context: (n/a)

The FileVault enable workflow does two primary things: first it sends a Configuration Profile to the device (containing the payloads for FileVault escrow, and deferred enablement, and an certificate for encryption). Then it polls the device with a `SecurityInfo` command waiting for the device (likely the end-user) to have enabled FileVault. Once this is done it escrows the FileVault PRK to the inventory system. The default polling is once a minute with a limit of 180 (in other words about 6 hours).

Note that the profile template can be customized. You'll first need to export the profile by using the API endpoint then re-upload your changed profile to the profile store *with the same name as the workflow*. The system will query the profile store every time the workflow starts first and will fallback to the built-in profile template if it is missing.

### FileVault Rotate Workflow

* Workflow name: `io.micromdm.wf.fvrotate.v1`
* Start value/context: (n/a)

The rotate FileVault workflow sends an MDM command to rotate the enrolled device's FileVault FDE Personal Recovery Key (PRK). It will retrieve the existing PRK from inventory subsystem in order to rotate the key. The new PRK will be escrowed back to inventory subsystem.

### Inventory Workflow

* Workflow name: `io.micromdm.wf.inventory.v1`
* Start value/context: (n/a)

The inventory workflow sends `DeviceInformation` and `SecurityInfo` commands to the enrollment to collect information from the host and store it in the inventory subsystem. As well the inventory workflow updates the inventory for any other `SecurityInfo` command that happens to be sent by any other workflow (as this command has no input to make it context-dependent).

### Profile Workflow

* Workflow name: `io.micromdm.wf.profile.v1`
* Start value/context: comma-separated list of profile names. removals prefixed with a minus/dash (-)
  * Example: `profile1,profile2,-profile3,profile4`

The profile workflow manages Configuration Profile "state" on an enrollment for the set of provided profile names. The workflow checks the already-installed profile identifiers and UUIDs to make sure the profiles are current and if not (or they are missing) installs them. The list of profiles is specified as a comma-separated list of profile names already stored in the profile subsystem. You can also specify profiles to be removed by prefixing them with a minus/dash (-) character.

For example, this start/context value:

```
dock,munki,-uakel,pppc
```

Would try to make sure that the profiles with the names of `dock`, `munki`, and `pppc` in the profile subsystem are installed (if they are not already) while making sure the `uakel` profile is removed (if it is installed).

### Lock Workflow

* Workflow name: `io.micromdm.wf.lock.v1`
* Start value/context: (n/a)

The lock workflow sends the device a lock command using a random PIN code that is escrowed to the inventory subsytem.

### Device Information Logger Workflow

* Workflow name: `io.micromdm.wf.devinfolog.v1`
* Start value/context: (n/a)

The "devinfolog" workflow sends the device a `DeviceInformation` MDM command with a few queries and simply logs the response. It has no other dependencies and offers an innocuous read-only workflow for testing or validation.
