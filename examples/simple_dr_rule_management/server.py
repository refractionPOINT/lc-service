import lcservice
import limacharlie
from arl import AuthenticatedResourceLocator as ARL

import os
import time
from threading import Lock
import yaml
import traceback

# This example uses GitHub REST API to access a private repo, download
# rules and mirror them on organizations that install the service.
# The service requires the following permissions:
# - dr.set.replicant
# - dr.del.replicant
# - dr.list.replicant
# It should also use the following Flairs for its permissions:
# - segment
# - lock
# - secret

# User access token is expected in the GITHUB_TOKEN environment variable.
GITHUB_TOKEN = os.environ.get( 'GITHUB_TOKEN' )
# Repo path, example: https://github.com/example-org/corp-dr-rules
GITHUB_ORG = 'example-org'
REPO_NAME = 'corp-dr-rules'

class ExampleService( lcservice.Service ):

    def onStartup( self ):
        self.rulesLock = Lock()
        self.svcRules = None
        self.lastRulesRefresh = 0
        self.refreshRulesEvery = 60 * 60 * 3

        self.downloadRules()

    def onOrgInstalled( self, lc, oid, request ):
        print( 'Org %s just subscribed.' % ( oid, ) )

        # First we apply our rules for the first time.
        # But once we're running, we'll sync every hour.
        self.every1HourPerOrg( lc, oid, request )

        return True

    def onOrgUninstalled( self, lc, oid, request ):
        self.log( "Goodbye %s" % ( oid, ) )

        # Remove all out detections.
        # We do this using a concurrent call to del_rule instead
        # of using a Sync just because the Sync doesn't allow you
        # to do them concurrently and if you have 100s of rules it
        # could take too long to remove them sequentially.
        for ruleName, result in self.parallelExecEx( lambda x: lc.del_rule( x, namespace = 'replicant' ),
                                                     { k: k for k in self.svcRules.keys() },
                                                     maxConcurrent = 5 ):
            # If an exception occured during a call, it will be
            # returned as the return value. So log it if it's an exception.
            if isinstance( result, Exception ):
                self.logCritical( "Failure deleting rule %s on %s: %s" % ( ruleName, oid, result ) )
        return True

    def every1HourPerOrg( self, lc, oid, request ):
        # Check if we have recent-enough version of the rules.
        with self.rulesLock:
            now = time.time()
            isNeedsRefresh = False
            if now > self.lastRulesRefresh + self.refreshRulesEvery:
                # We haven't refreshed in a while, download the
                # latest version.
                self.lastRulesRefresh = now
                isNeedsRefresh = True

        if isNeedsRefresh:
            self.log( "Rules too old, refreshing." )
            self.downloadRules()

            # We don't know how long it took to download the rules
            # so instead of risking going over time, we will signal
            # LimaCharlie to retry this action (we should have up
            # to date rules now).
            return None

        # At this point we have up to date rules.
        sync = limacharlie.Sync( manager = lc )

        # We use the Sync component of the SDK to take care
        # of updating rules that need it and remove old ones.
        # So we structure the rules like an LC config file
        # since it's what Sync expects.
        rules = {
            'rules' : self.svcRules,
        }

        # Check that the rules are applied. The isForce = True
        # means we also want to remove rules in prod no longer
        # in the yaml file we got from the repo, so a mirror.
        sync.pushRules( rules, isForce = True )

        # Success, don't retry.
        return True

    def onDetection( self, lc, oid, request ):
        detection = request.data
        self.log( "Received a detection %s on %s" % ( detection[ 'cat' ], oid ), data = detection )

        # Success, don't retry.
        return True

    def downloadRules( self ):
        newRules = {}
        newDetections = set()

        # We assume all D&R rules are in detections.yaml and all internal
        # lookup resources those rules use are each in a "resources/RESOURCE_NAME"
        # sub-directory in the repo.
        with ARL( '[github,%s/%s,token,%s]' % ( GITHUB_ORG, REPO_NAME, GITHUB_TOKEN, ), maxConcurrent = 5 ) as r:
            for fileName, content in r:
                # The detections are in a single file "detections.yaml".
                # like: ruleName => {detect => ..., respond => ...}
                if 'detections.yaml' == fileName:
                    try:
                        newRules = yaml.safe_load( content )
                    except:
                        raise Exception( "failed to parse yaml from rules file %s: %s" % ( fileName, traceback.format_exc() ) )

                # Resources are in a "resources/" directory.
                if fileName.startswith( 'resources/' ):
                    # This is a resource, use the filename without extension as name.
                    resourceName = fileName.split( '/', 1 )[ 1 ]
                    resourceName = resourceName[ : resourceName.rfind( '.' ) ]
                    # We assume all resources are lookups.
                    self.publishResource( resourceName, 'lookup', content )

        for ruleName, rule in newRules.items():
            # Make sure the rule goes in the "replicant" namespace. This way
            # we don't need to set the namespace in the yaml file.
            rule[ 'namespace' ] = 'replicant'

            for drResponse in rule[ 'respond' ]:
                if 'report' == drResponse[ 'action' ]:
                    # We want to be notified for all rule reports.
                    newDetections.add( drResponse[ 'name' ] )

        # Update the rules in effect.
        self.svcRules = newRules

        # Make sure we're subscribed to all the notifications.
        for detection in newDetections:
            self.subscribeToDetect( detection )

def main():
    # Bind to a potentially dynamic port for things like Google Cloud Run
    # or other clusters.
    port = int( os.environ.get( 'PORT', '80' ) )
    print( 'Starting on port %s' % ( port, ) )

    # Create an instance of our service with our service name and the
    # shared secret with LimaCharlie to verify the origin of the data.
    svc = ExampleService( 'example-service',
                          os.environ.get( 'LC_SHARED_SECRET' ),
                          os.environ.get( 'DO_TRACE', False ) )

    # Start serving it using CherryPy as an HTTP server.
    lcservice.servers.ServeCherryPy( svc, interface = '0.0.0.0', port = port )

    print( 'Shut down' )

if __name__ == '__main__':
    main()