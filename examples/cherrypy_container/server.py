import lcservice
import os
import limacharlie
import yaml
import json

class ExampleService( lcservice.Service ):

    def onStartup( self ):
        # This is a core set of rules we want to
        # ensure are always installed.
        self.svcRules = yaml.safe_load( '''
            test-rule-1:
              detect:
                op: contains
                path: event/FILE_PATH
                value: \\recycler\\
                case sensitive: false
                event: NEW_PROCESS
              respond:
                - action: report
                  name: detected-test-rule-1
            test-rule-2:
              detect:
                op: ends with
                path: event/FILE_PATH
                value: \\evil.exe
                case sensitive: false
                event: NEW_PROCESS
              respond:
                - action: report
                  name: detected-test-rule-2
        ''' )

        # Make sure we get notifications from those.
        self.subscribeToDetect( 'detected-test-rule-1' )
        self.subscribeToDetect( 'detected-test-rule-2' )

    def onOrgInstalled( self, lc, oid, request ):
        print( 'Org %s just subscribed.' % ( oid, ) )

        # First we apply our rules for the first time.
        self.every1HourPerOrg( lc, oid, request )

        return True

    def every1HourPerOrg( self, lc, oid, request ):
        sync = limacharlie.Sync( manager = lc )

        rules = {
            'rules' : self.svcRules,
        }

        # Check that the rules are applied.
        sync.pushRules( rules )

        return True

    def onOrgUninstalled( self, lc, oid, request ):
        self.log( "Goodbye %s" % ( oid, ) )

        # Remove all out detections.
        for ruleName in self.svcRules.keys():
            lc.del_rule( ruleName )

        return True

    def onDetection( self, lc, oid, request ):
        self.log( "Received a detection: %s" % ( json.dumps( request.data ), ) )

        return True

    def every24HourPerSensor( self, lc, oid, request ):
        sid = request.data[ 'sid' ]

        # We will query the sensor live.
        lc.make_interactive()

        try:
            # Let's list the pakages installed on this box.
            response = lc.sensor( sid ).simpleRequest( [ 'os_packages' ] )
            if response is None:
                return False
            packages = response[ 'event' ].get( 'PACKAGES', [] )
            self.log( "Sensor %s has %s packages installed." % ( sid, len( packages ) ) )
        finally:
            # Being interactive with a sensor takes some resources, so we
            # will shutdown cleanly as soon as we can.
            lc.shutdown()

def main():
    # Bind to a potentially dynamic port for things like Google Cloud Run
    # or other clusters.
    port = int( os.environ.get( 'PORT', '80' ) )
    print( 'Starting on port %s' % ( port, ) )

    # Create an instance of our service with our service name and the
    # shared secret with LimaCharlie to verify the origin of the data.
    svc = ExampleService( 'example-service', 'my-secret' )

    # Start serving it using CherryPy as an HTTP server.
    lcservice.servers.ServeCherryPy( svc, interface = '0.0.0.0', port = port )

    print( 'Shut down' )

if __name__ == '__main__':
    main()