# LimaCharlie.io Services

![LimaCharlie.io](https://storage.googleapis.com/limacharlie-io/logo_fast_glitch.gif)

Reference implementation of the LimaCharlie Service protocol.

# Reference Implementation
LimaCharlie services implement a publicly documented JSON/HTTP based protocol.
This means you may implement a service that speaks this protocol in whaterver
language or architecture you like.

That being said, to simplify the life of users and to describe more concretely
the expectation of a working service, LimaCharlie publishes an open source
Reference Implementation (RI) in Python.

API Documentation: https://lc-service.readthedocs.io/

## Transports
The RI supports two transports out of the box: standalone container based on
the CherryPy project for an HTTP server, and a Google Cloud Function compatible
server to deploy without containers or infrastructure. Writing transports is easy
so feel free to suggest new ones.

***Cloud Function Transport*** is currently having an issue with the initialization
and monkey patching of gevent in the Cloud Function environment.

## Using

The RI is structure so that all you have to do is inherit from the main Service class:

```python
class MyService ( lcservice.Service ):
```

and implement one or more callback functions:

* `onStartup`: called once when an instance of the service starts.
* `onShutdown`: called once when an instance of the service stops.
* `onOrgInstalled`: an organization installs your service.
* `onOrgUninstalled`: an organization uninstalls your service.
* `onDetection`: a detection your service subscribes to occured.
* `onRequest`: an ad-hoc request for your service is received.
* `onDeploymentEvent`: a deployment event (like sensor enrollment, over-quota, etc) is received from the cloud.
* `onLogEvent`: a new log has been ingested in the LimaCharlie cloud.
* `onServiceError`: the LimaCharlie encountered an error while dealing with your service.

as well as any number of callbacks from a series of cron-like functions provided
with various granularity (global, per organization or per sensor).

* `every1HourPerOrg`
* `every3HourPerOrg`
* `every12HourPerOrg`
* `every24HourPerOrg`
* `every7DayPerOrg`
* `every30DayPerOrg`
* `every1HourGlobally`
* `every3HourGlobally`
* `every12HourGlobally`
* `every24HourGlobally`
* `every7DayGlobally`
* `every30DayGlobally`
* `every24HourPerSensor`
* `every7DayPerSensor`
* `every30DayPerSensor`

All the callbacks receive the same arguments `( lc, oid, request )`:

* `lc`: an instance of the [LimaCharlie SDK](https://github.com/refractionPOINT/python-limacharlie/) pre-authenticated for the relevant organization.
* `oid` the Organization ID of the relevant organization.
* `request`: a `lcservice.service.Request` object containing a `eventType`, `messageId` and `data` properties.
* for `*PerSensor` callbacks, the `request` contains a `sid` value for a specific sensor.

Each callback returns one of 4 values:

* `True`: indicates the callback was successful.
* `False`: indicates the callback was not successful, but the request should NOT be retried.
* `None`: indicaes the callback was not successful, but the request SHOULD be retried.
* `self.response( isSuccess = True, isDoRetry = False, data = {}, error = None, jobs = [] )`: to customize the behavior requested of the LimaCharlie platform.

In addition to the main lifecycle callbacks, some functions are available to simplify
some management tasks for your service.

* `subscribeToDetect( detectName )`: allows you specify the names of detections you would like to receive notifications from in the `onDetection` callback.
* `publishResource( resourName, resourceCategory, resourceData )`: allows you to make available to LimaCharlie resources private to your service, like a `lookop` for example. You can refer to them as `lcr://service/<serviceName>/<resourceName>`.
* `setRequestParameters( parameters )`: allows you to specify what parameters are accepted in a request to your service, see the protocol section below for an exact format.

Many helper functions are also provided for your convenience like:

* `log()`
* `logCritical()`
* `delay()` and `schedule()`
* `parallelExec()`

## Deploying

Deploying is slightly different depending on the transport chose.

For the CherryPy container based transport, deployment is as simple as running
the container. For the sake of handling less infrastructure we recommend Using
something like [Google Cloud Run](https://cloud.google.com/run/docs/).

A pre-built base image container is available [here](https://hub.docker.com/r/refractionpoint/lc-service):

```
FROM refractionpoint/lc-service:latest
```

See our example service [here](examples/).

## Examples

Some ready to deploy examples can be found [here](examples/).

### List Packages For New Sensors

```python
class MyService( lcservice.Service ):
    def onDeploymentEvent( self, lc, oid, request ):
        # We only care about enrollments of new sensors.
        if 'enrollment' != request.data[ 'routing' ]:
            return True

        sensorId = request.data[ 'routing' ][ 'sid' ]

        sensor = lc.sensor( sensorId )

        if not sensor.isOnline():
            # It must have disconnected already.
            return False

        # We will interact in real-time with the host.
        lc.make_interactive()

        response = sensor.simpleRequest( [ 'os_packages' ] )
        if response is None:
            # It might have disconnected.
            return False

        self.log( "New sensor %s has packages: %s" % ( sensorId, response[ 'event' ] ) )
```

## Best Practices

### Resource Usage
If your service creates [D&R rules](http://doc.limacharlie.io/en/master/dr/), it is
recommended that those rules not rely on other external resource (like `lcr://lookup/something`)
because organizations who install your service may not be subscribed to those resources
which in turn will mean your rules will fail.

There are two solutions to this problem:

1. Have your service request the `billing.ctrl` permission and then have the service
register the organization to this external resource.
1. (Recommended) Make the resource an internal to the service (using `publishResource` as mentioned above)
and make your rules use the internal resource (which do NOT require organization detect_subscription)
like `lcr://service/<your-service-name>/<your-internal-resource-name>`. This way your
service is self-contained and does not require external resources.

### Permissions Usage
When creating a service record in LimaCharlie, you will have a choice of which
permissions the service requests to begin operation. Although you can enable
all permissions, it's generally not advised as it grants a wide number of critical
permissions.

There are no limits on which permissions you request for private services. However
if you intend to add your service publicly through the marketplace (and monetize it)
the LimaCharlie team is likely going to request to revisit with you your usage of
any permissions that are more sensitive (like `user.ctrl` for example).

In addition to permissions, you may also chose various pieces of [Flair](https://doc.limacharlie.io/en/master/api_keys/#flair)
your service requests to apply to its API usage. Although again here you have full
control, we generally recommend you enable the following:

* `lock`: this ensures that resources your service creates don't get overwritten by other services or users.
* `secret`: this ensures that the content of the resources your service creates are not visible to others. It's not as critical but if you intend to install proprietary [Detection & Response rules](https://doc.limacharlie.io/en/master/dr/) you likely want it.
* `segment`: this ensures that your services does not see any resources it has not created itself. This helps ensure your service doesn't delete other services' resources as well as maintain general privacy.

## Tips
The following are general tips to know when developing new services.

### Start on Simulator
You can use the Simulator like: `python -m lcservice.simulator` before trying
to standup your service live with LimaCharlie. This makes it faster to test
some of the functionality. By setting the shared secret of your service to `None`
the origination of requests is not checked so you can use a simple `curl` as well.

### Adding Live Service
When adding a new service to LimaCharlie, it may take up to ~5 minutes for it
to become available on all LimaCharlie data-centers. Trying to subscribe to
it before it's available may result in odd behavior. If you encounter those, simply
un-register and re-register your Organization.

### Permissions Changes
If you change the permissions for a service after it has been deployed and used by
an organization, the new permissions do NOT propagate to existing organizations. To
force the new permissions to take effect, un-register and re-register the organizations.
Also note that because JWTs may be cached within LimaCharlie, it's possible for your new
permissions to not be in effect for up to an hour. This means you should take care at
figuring out the permissions you require ahead of time.

### Detection Subscription Changes
A service may register to receive some detections from LimaCharlie. That list of
detection of interest is updated at recurring interval in LimaCharlie and may take
up to 5 minutes to update.

### Tasking Sensors
If you need to task a sensor, generally favor using a combination of investigation_id
and D&R rule (as seen in the `job_usage` example) if the tasking is a core part of
what the service is doing. You can use `sensor.simpleRequest()` for doing sporadic
requests, but doing so (and `lc.make_interactive()`) has a few drawbacks:

* Tasking a sensor and getting a reply can be slow in some cases, leading to timeouts of your service.
* Tasking sensors interactively requires additional `output.*` permissions.
* Interactive tasking has a significant overhead.

Building your service flow around detections and tracking state using investigation_id
will allow your service to scale better.

# Protocol
LimaCharlie Services rely entirely on response to REST calls (webhooks)
from LimaCharlie, making passive deployments through AWS Lambda, GCP
Cloud Functions or GCP Cloud Run possible.

Each HTTP POST contains JSON. The request content and responses are entirely
based on JSON content and not HTTP status codes (to possibly enable other
transport protocols if needed in the future).

A request contains the following data:

* `version`: this is the version of the protocol spoken by LimaCharlie.
* `oid`: the Organization ID this call is about (if any).
* `mid`: a unique Message ID which can be used to perform idempotent operations.
* `deadline`: a timestamp of how long LimaCharlie is willing to wait for this call.
* `jwt`: a JWT for the given `oid`, valid for AT LEAST 30 minutes, with the requester permissions for the service.
* `etype`: an event type (described below).
* `data`: arbitrary JSON, content depends on the `etype`.

A response from the service, also JSON is expected to have the following format:

* `success`: a boolean indicating whether the call was successful.
* `retry`: if `success` was `false`, should LimaCharlie attempt to re-deliver this message.
* `data`: arbitrary JSON, content based on the `etype` in the request.
* `error`: optionally an error to report to the organization.
* `jobs`: optionally new jobs or updates to existing jobs.

Most requests will have a deadline of +590s in the future. This may mean that longer
operations will not fit in that deadline. You should either delay execution, parallelize
or split up the execution in more granular `etype` events like per-sensor events.

More complex `etype` will be available in the future to allow services to request
extensions or record longer running jobs.

## Event Types

### health
This call is LimaCharlie requesting a status from your service. The following information
is expected to be returned in the data.

* `version`: the version of the protocol spoken by the service.
* `start_time`: a timestamp of when this service instance started.
* `calls_in_progress`: the number of of calls in progress on this instance.
* `mtd`: a JSON dictionary with the following keys descibed below.

**Metadata** found in the `mtd` key describes what subset of the protocol is
used by this service:

* `detect_subscriptions`: a list of detections this service would like to receive for organizations subscribed.
* `callbacks`: the list of `etypes` supported/used by this service (telling LimaCharlie not to bother with the others).
* `request_params`: a dictionary describing supported parameters in requests defined to this service, full definition below.

**Request Parameters**
This dictionary should be of the form `param_name => { type, desc }`. These definitions
will be used by LimaCharlie to construct simplified request user interfaces to your service.
Your service should still do full validation of parameters passed to it.

The `type` is one of `int`, `float`, `str`, `bool`.

The `desc` should be a short description of the purpose and interpretation of the parameter.

Example for a fictional payload detonation service:
```json
{
  "action": {
    "type": "str",
    "desc": "the action to take, one of 'set' or 'get'.",
  },
  "api_key": {
    "type": "str",
    "desc": "the api key to use when requesting a payload detonation."
  },
  "retention": {
    "type": "int",
    "desc": "the number of days to set when ingesting detonation artifacts."
  }
}
```

### org_install
Indicates that a new organization has installed the service (subscribed).
Setup the organization with all the required configurations here.

### org_uninstall
Indicates that an organization has uninstalled the service. Remove all configurations
that were made on that organization here. All traces of your service should be gone.

### detection
Called when a detection this service subscribes to (see `detect_subscriptions`) occurs.

### request
Will support interactive requests by users within the organization for ad-hoc functionality
of your service like running jobs on specific hosts etc. Not yet implemented.

### get_resource
Called when a resource that is internal to the service is requested by LimaCharlie.
The data in the request includes:

* `resource`: the name of the resource requested.
* `is_include_data`: if true, the actual resource content is requested, otherwise only the hash.

The expected response by LimaCharlie has the following data elements:

* `hash`: the sha256 of the content of the resource, used to determine if LimaCharlie needs to refresh it.
* `res_cat`: the resource category (like `lookup` or `detect`) of the resource returned.
* `res_data`: if the data was requested, this is the `base64(data)`.

### org_per_*
Convenience cron-like event. The LimaCharlie cloud emits those events at recurring
interval on a per-organization basis so you don't have to keep track of timing or
setup cron jobs.

* `org_per_1h`
* `org_per_3h`
* `org_per_12h`
* `org_per_24h`
* `org_per_7d`
* `org_per_30d`

### once_per_*
Convenience cron-like event. The LimaCharlie cloud emits those events at recurring
interval on a per-service basis so you don't have to keep track of timing or
setup cron jobs.

* `once_per_1h`
* `once_per_3h`
* `once_per_12h`
* `once_per_24h`
* `once_per_7d`
* `once_per_30d`

### sensor_per_*
Convenience cron-like event. The LimaCharlie cloud emits those events at recurring
interval on a per-sensor basis so you don't have to keep track of timing or
setup cron jobs.

* `sensor_per_1h`
* `sensor_per_3h`
* `sensor_per_12h`
* `sensor_per_24h`
* `sensor_per_7d`
* `sensor_per_30d`

### deployment_event
Called when a deployment event occurs in an organization with the service installed.
The `data` component will contain a `routing` and `event` component similarly to the
deployment events in a LimaCharlie [Output](https://doc.limacharlie.io/en/master/outputs/).

### log_event
Called when a log has been ingested in LimaCharlie. The `data` component will contain a `routing` and `event` component similarly to the
deployment events in a LimaCharlie [Output](https://doc.limacharlie.io/en/master/outputs/).

### service_error
Called when the LimaCharlie cloud encounters an error while dealing with your service.