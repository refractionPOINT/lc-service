# lc-service
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

Each callback returns one of 4 values:

* `True`: indicates the callback was successful.
* `False`: indicates the callback was not successful, but the request should NOT be retried.
* `None`: indicaes the callback was not successful, but the request SHOULD be retried.
* `self.response( isSuccess = True, isDoRetry = False, data = {} )`: to customize the behavior requested of the LimaCharlie platform.

In addition to the main lifecycle callbacks, some functions are available to simplify
some management tasks for your service.

* `subscribeToDetect( detectName )`: allows you specify the names of detections you would like to receive notifications from in the `onDetection` callback.
* `publishResource( resourName, resourceCategory, resourceData )`: allows you to make available to LimaCharlie resources private to your service, like a `lookop` for example. You can refer to them as `lcr://service/<serviceName>/<resourceName>`.

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

Most requests will have a deadline of +60s in the future. This may mean that longer
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