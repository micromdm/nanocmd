# NanoCMD Quick Start Guide

This quickstart guide is intended to quickly get a functioning NanoCMD instance up and running and able to use some of the included workflows.

## Requirements

* A functioning NanoMDM or MicroMDM (v1.9.0 or later) server.
  * You'll need to know the URLs of the command submission and APNs push API endpoints.
  * For [NanoMDM](https://github.com/micromdm/nanomdm/blob/main/docs/operations-guide.md#enqueue) this is usally `/v1/enqueue/` and `/v1/push/` endpoints.
  * For [MicroMDM](https://github.com/micromdm/micromdm/blob/main/docs/user-guide/api-and-webhooks.md#schedule-raw-commands-with-the-api) this is usually `/v1/commands` and `/push`.
* An enrolled macOS device in that MDM server.
  * It doesn't have to be a macOS device but for the demo below it makes things easier.
  * You'll need to know the device's [enrollment ID](https://github.com/micromdm/nanomdm/blob/main/docs/operations-guide.md#enrollment-ids) — usually its UDID.

## Setup and start server

### NanoCMD

You'll need the NanoCMD server to start running it, of course. You can fetch it from the [NanoCMD GitHub releases page](https://github.com/micromdm/nanocmd/releases). You can also build it from source if you prefer but that's outside the scope of this document.

Next you'll need to run it and point it at your MDM server. Here's an example invocation for running against a NanoMDM server:

```sh
./nanocmd-darwin-amd64 \
  -api supersecret \
  -enqueue-api supersecretNano \
  -enqueue-url 'http://[::1]:9000/v1/enqueue/' \
  -push-url 'http://[::1]:9000/v1/push/' \
  -debug
```

You can review the [operations guide](../docs/operations-guide.md) for the full command-line flags but we'll briefly review them here:

* `-api` configures the API password for NanoCMD.
* `-enqueue-url` is the URL that commands are submitted to NanoMDM.
* `-enqueue-api` is the API password for your NanoMDM command enqueue API.
* `-push-url` is the URL that APNs pushes are submitted to NanoMDM.
* `-debug` turns on additional debug logging.

If we wanted to run it against MicroMDM that might look like this:

```sh
./nanocmd-darwin-amd64 \
  -api supersecret \
  -enqueue-api supersecretMicro \
  -enqueue-url 'http://[::1]:8080/v1/commands/' \
  -push-url 'http://[::1]:8080/push/' \
  -micromdm \
  -debug
```

Note the changed URLs and the additional flag:

* `-micromdm` turns on the ability to talk to MicroMDM servers.

With either server the operation of NanoCMD should be the same. Once we start NanoMDM you'll see some output. One of the lines shoud look similar to this:

```sh
ts=2023-05-30T15:01:46-07:00 level=info msg=starting server listen=:9003 caller=main.go:159
```

Indicating to us that the NanoCMD server started and is listening on port 9003.

### NanoMDM (or MicroMDM)

NanoMDM (or MicroMDM) will need to be pointed "back" at NanoCMD's webhook URL handler. For NanoMDM you'll need to use [the `-webhook-url` flag](https://github.com/micromdm/nanomdm/blob/main/docs/operations-guide.md#-webhook-url-string) when starting NanoMDM. For MicroMDM you'll need to use [the `-command-webhook-url` flag](https://github.com/micromdm/micromdm/blob/main/docs/user-guide/api-and-webhooks.md#configure-a-webhook-url-to-process-device-events). For example if you started NanoMDM like this:

```sh
./nanomdm-darwin-amd64 -ca ca.pem -api supersecretNano -debug
```

You'll need to point it at NanoCMD like so:

```sh
./nanomdm-darwin-amd64 -ca ca.pem -api supersecretNano -debug -webhook-url 'http://[::1]:9003/webhook'
```

Simlar for MicroMDM.

Good, you should now have your MDM and NanoCMD pointed at each other. Let's have some fun with it!

## First workflow: inventory

For the rest of this guide, let's assume our device's enrollment ID is `FF269FDC-7A93-5F12-A4B7-09923F0D1F7F`. Also in many places the JSON output may look nice and formatted — I've taken the liberty of running it through `jq .` just so it's easier to read here. You're welcome to do that, too, but it may make errors harder to troubleshoot with the `curl` calls.

Let's check if there is any inventory data already. There shouldn't be, but let's make sure:

```sh
$ curl -u nanocmd:supersecret 'http://[::1]:9003/v1/inventory?id=FF269FDC-7A93-5F12-A4B7-09923F0D1F7F'
{}
```

Nothing returned, that's what we expected. Now, let's start the inventory workflow for this ID:

```sh
$ curl -u nanocmd:supersecret -X POST 'http://[::1]:9003/v1/workflow/io.micromdm.wf.inventory.v1/start?id=FF269FDC-7A93-5F12-A4B7-09923F0D1F7F'
{"instance_id":"d4f8c9c4-a8ef-4dd9-99e0-38dca53e60f4"}
```

You can see we returned an instance ID from starting this workflow. I'm sure you also saw *a lot* of output in the NanoCMD logs with all of the debug logging we enabled. The most important one we're looking for, perhaps, is this one:

```
ts=2023-05-30T15:26:29-07:00 level=debug service=engine trace_id=547000a1cebaac11 command_uuid=c146d3d5-0d75-4b28-8793-ad93e24f43f3 id=FF269FDC-7A93-5F12-A4B7-09923F0D1F7F engine_command=true request_type=SecurityInfo command_completed=true step_completed=true workflow_name=io.micromdm.wf.inventory.v1 instance_id=d4f8c9c4-a8ef-4dd9-99e0-38dca53e60f4 msg=completed workflow step caller=engine.go:371
```

Which indicates our the step for this instance ID completed for this enrollment ID (`step_completed=true`). This means that all commands that were enqueued as part of the initial step for this workflow completed.

So, let's check our inventory using the same query we issued above:

```sh
$ curl -u nanocmd:supersecret 'http://[::1]:9003/v1/inventory?id=FF269FDC-7A93-5F12-A4B7-09923F0D1F7F' | jq .
{
  "FF269FDC-7A93-5F12-A4B7-09923F0D1F7F": {
    "apple_silicon": true,
    "build_version": "22E261",
    "device_name": "Laika’s Virtual Machine",
    "ethernet_mac": "76:f1:14:93:fc:00",
    "fde_enabled": false,
    "has_battery": false,
    "last_source": "DeviceInformation",
    "model": "VirtualMac2,1",
    "model_name": "Virtual Machine",
    "modified": "2023-05-30T15:26:29.157322-07:00",
    "os_version": "13.3.1",
    "serial_number": "ZRMXJQTTFX",
    "sip_enabled": true,
    "supports_lom": false
  }
}
```

Ah, that's better. We can see that we populated a bunch of properties for this device from both `DeviceInformation` and `SecurityInfo` MDM commands. This data is persisted in the inventory subsystem storage for this device and is available whether the device is online or not.

We can run this workflow any time we want to update the inventory stored here. Attributes will get overwritten from the newer command responses.

## Second workflow: profiles

With MicroMDM or NanoMDM it's pretty easy to individually install (or remove) profiles, of course. You just send the relevant commands. However what if we want to get a bit more... stateful, or even (gasp) idempotent? NanoCMD's profile workflow may be able to help. It can install or remove profiles based on the profiles already installed by querying them first. Let's give it a try.

First, we need to upload a profile. I like to use a simple Dock profile that changes the Dock orientation on macOS to the left because it's gives instant visual feedback when it gets installed. [Here is an example](https://gist.github.com/jessepeterson/27d39e8cc4d7ed81773b0a5e2cdc01f5) that I've call `dockleft.mobileconfig`.

Before we upload, let's check if we have this profile in NanoCMD already:

```sh
$ curl -u nanocmd:supersecret 'http://[::1]:9003/v1/profiles?name=dockleft'
{
  "error": "profile not found for dockleft: profile not found"
}
```

Okay, not uploaded yet. As we expected. Let's upload it!

```sh
$ curl -u nanocmd:supersecret -w "%{http_code}\n" -T ~/Desktop/dockleft.mobileconfig 'http://[::1]:9003/v1/profile/dockleft'
204
```

Here we uploaded the `dockleft.mobileconfig` on my desktop to the `/v1/profile/dockleft` URL — the 204 (No Content) status update is expected. The last part of that URL is the name of profile in the profile subsystem storage. Now, if we query our profiles like we did before:

```sh
$ curl -u nanocmd:supersecret 'http://[::1]:9003/v1/profiles?name=dockleft' | jq .
{
  "dockleft": {
    "identifier": "com.example.dockleft",
    "uuid": "D0C38014-4DBB-4F19-A23F-2768FA2246AE"
  }
}
```

We can see that our "dockleft" profile is uploaded with a specific identifier and UUID which was in the profile. We can also retrieve this profile from the store as well:

```sh
$ curl -u nanocmd:supersecret 'http://[::1]:9003/v1/profile/dockleft' | head -5
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>PayloadContent</key>
```

Which is our raw profile contents.

Okay, so we have a profile in the store, how do we get it to install? By using our workflow, of course! Let's kickoff the profile workflow specifying this profile name to install:

```sh
$ curl -u nanocmd:supersecret -X POST 'http://[::1]:9003/v1/workflow/io.micromdm.wf.profile.v1/start?id=FF269FDC-7A93-5F12-A4B7-09923F0D1F7F&context=dockleft'
{"instance_id":"3929e3a7-8ae9-473c-96ba-1938df998b4a"}
```

Notice the `context=` parameter at the end there. If all went to plan you should have seen the Dock disappear and reappear on the left hand side of the screen on macOS.

More to the point though the workflow queried the installed profiles to determine if it needed to install this profile, found that it did, and sent that MDM command. We can see this in the logs with lines like `request_type=ProfileList command_completed=true`

If we run that exact workflow again the device should query the profiles and find that, because its identifier and UUID have not changed, it won't need to be installed. And indeed that's what happens, telling us in the logs: `msg=no profiles to install or remove after profile list`.

Now, what if we want to remove that profile? No problem. Prefix the profile name with a minus/dash (-) sign like so:

```sh
$ curl -u nanocmd:supersecret -X POST 'http://[::1]:9003/v1/workflow/io.micromdm.wf.profile.v1/start?id=FF269FDC-7A93-5F12-A4B7-09923F0D1F7F&context=-dockleft'
{"instance_id":"3929e3a7-8ae9-473c-96ba-1938df998b4a"}
```

The profile will then be removed from the system. We see lines in the logs like: `request_type=ProfileList command_completed=true step_completed=true`. Similarly to the install case, if we run this exact command again, it won't try to remove the profile (because it isn't installed).

Now, that's fun for single profiles. But the workflow supports multiple installs and removals, too. If you've uploaded a number of profiles you can specify them all by supplying them in the context separated by commas. For example:

```
dockleft,munki,-uakel,pppc
```

The profile workflow will work out what those profile identifiers are (from the profile subsystem storage), query the device for its list of profiles, and determine what MDM commands need to happen to make the installed profiles match the specified state — including, as above, taking no action if the device already has the correct matching set of profiles.

Combining the fact that you can start a workflow for multiple enrollment IDs you can have an invocation like this:

```sh
$ curl -u nanocmd:supersecret -X POST 'http://[::1]:9003/v1/workflow/io.micromdm.wf.profile.v1/start?id=DEV1&id=DEV2&id=DEV3&context=dockleft,munki,-uakel,pppc'
{"instance_id":"3929e3a7-8ae9-473c-96ba-1938df998b4a"}
```

Which would manage the state of those four profiles (`dockleft`, `munki`, `uakel`,  and `pppc`) on those three devices (`DEV1`, `DEV2`, and `DEV3`) and be smart about only installing (or removing) profiles on the devices that need to be.

As well if you upload a different version of the profile (and its UUID changes) then you don't need to change your workflow invocation. It will actively query the profile store to figure out the newest version and issue the install profile command just for that change.

## Next steps

Those are two example workflows. Here's a few ideas on where to proceed next:

* Read the [Operations Guide](../docs/operations-guide.md) for more details on configuration, troubleshooting, etc.
* Try other workflows! See the operations guide for documentation.
  * Command Plans — groups of MDM commands intended for installation
  * FileVault enable and rotate — enables deferred FileVault, polls device to escrow the PSK, and can rotate PSKs.
* Configure Event Subscriptions to e.g. start workflows on device enrollment. See the operations guide for documentation.
* Configure a proper deployment
  * Behind HTTPS/proxies
  * Behind firewalls or in a private cloud/VPC
  * In a container environment like Docker, Kubernetes, etc. or even just running as a service with systemctl.
