import lcservice
import os
import limacharlie
import yaml
import json
from lcservice.jobs import Job
from lcservice.jobs import YamlData

class ExampleService( lcservice.Service ):

    def onStartup( self ):
        self.completionInvId = 'package_survey_request'
        self.completionDetection = '__package_survey_done'
        # This is a core set of rules we want to
        # ensure are always installed.
        # We set a rule waiting for successful responses
        # to OS_PACKAGES we've issued with a specific
        # investigation ID we use to track the responses.
        self.svcRules = yaml.safe_load( '''
            package-survey:
              namespace: replicant
              detect:
                op: starts with
                path: routing/investigation_id
                value: %s
                event: OS_PACKAGES_REP
              respond:
                - action: report
                  name: %s
        ''' % ( self.completionInvId, self.completionDetection ) )

        # Make sure we get notifications from those.
        self.subscribeToDetect( self.completionDetection )

    def onOrgInstalled( self, lc, oid, request ):
        print( 'Org %s just subscribed.' % ( oid, ) )

        # First we apply our rules for the first time.
        self.every1HourPerOrg( lc, oid, request )

        return True

    def onOrgUninstalled( self, lc, oid, request ):
        self.log( "Goodbye %s" % ( oid, ) )

        # Remove all out detections.
        for ruleName in self.svcRules.keys():
            try:
                lc.del_rule( ruleName )
            except Exception as e:
                self.logCritical( "Failure deleting rule %s on %s: %s" % ( ruleName, oid, e ) )

        return True

    def every1HourPerOrg( self, lc, oid, request ):
        sync = limacharlie.Sync( manager = lc )

        rules = {
            'rules' : self.svcRules,
        }

        # Check that the rules are applied.
        sync.pushRules( rules )

        return True

    def every30DayPerSensor( self, lc, oid, request ):
        # Per sensor callbacks receive the SID in the request data.
        sid = request.data[ 'sid' ]

        # Only attempt to survey if the box is online.
        sensor = lc.sensor( sid )
        if sensor.isOnline():
            # Ok, we'll actually go and survey, create a job
            # and set the cause. Also specify that this SID is
            # involved in this job.
            job = Job()
            job.setCause( "Starting packages survey on sensor." )
            job.addSensor( sid )

            # We set the investigation ID we use to track responses. The
            # first part is a constant prefix to track responses relating to
            # this service, the second part is just the JobId.
            trackingInvId = '%s/%s' % ( self.completionInvId, job.getId() )
            try:
                sensor.task( [ 'os_packages' ], inv_id = trackingInvId )
            except Exception as e:
                self.logCritical( "Error surveying %s on %s: %s" % ( sid, oid, str( e ) ) )
            else:
                # Survey went out fine, let's update our job to the user.
                return self.response( isSuccess = True, jobs = [ job ] )

        return True

    def onDetection( self, lc, oid, request ):
        self.log( "Received a detection: %s" % ( json.dumps( request.data ), ) )

        # A survey completed, we'll use the JobID we stuffed as the last
        # part of the investigation ID to update the job.
        _, jobId = request.data[ 'routing' ][ 'investigation_id' ].split( '/' )

        # We want to update our pre-existing job.
        job = Job( jobId )

        # Narrate() allows us to explain our results. We can add
        # attachments to it to expose some of the data we collected.
        job.narrate( message = 'Found the following packages:', attachments = [
            YamlData( 'Packages', request.data[ 'detect' ][ 'event' ] ),
        ] )

        # This job is done.
        job.close()

        return self.response( isSuccess = True, jobs = [ job ] )

    def onNewSensor( self, lc, oid, request ):
        # New sensors get a survey right away.
        return self.every30DayPerSensor( lc, oid, request )

    def onRequest( self, lc, oid, request ):
        # This is just a manual trigger for a survey.
        return self.every30DayPerSensor( lc, oid, request )

def main():
    # Bind to a potentially dynamic port for things like Google Cloud Run
    # or other clusters.
    port = int( os.environ.get( 'PORT', '80' ) )
    print( 'Starting on port %s' % ( port, ) )

    # Create an instance of our service with our service name and the
    # shared secret with LimaCharlie to verify the origin of the data.
    svc = ExampleService( 'example-service',
                          'my-secret',
                          isTraceComms = os.environ.get( 'DO_TRACE', False ) )

    # Start serving it using CherryPy as an HTTP server.
    lcservice.servers.ServeCherryPy( svc, interface = '0.0.0.0', port = port )

    print( 'Shut down' )

if __name__ == '__main__':
    main()