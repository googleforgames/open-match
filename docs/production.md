During alpha, please do not use Open Match as-is in production.  To develop against it, please see the [development guide](development.md).

# "Productionizing" a deployment
Here are some steps that should be taken to productionize your Open Match deployment before exposing it to live public traffic.  Some of these overlap with best practices for [productionizing Kubernetes](https://cloud.google.com/blog/products/gcp/exploring-container-security-running-a-tight-ship-with-kubernetes-engine-1-10) or cloud infrastructure more generally. We will work to make as many of these into the default deployment strategy for Open Match as possible, going forward.
**This is not an exhaustive list and addressing the items in this document alone shouldn't be considered sufficient.  Every game is different and will have different production needs.**

## Kubernetes
All the usual guidance around hardening and securing Kubernetes are applicable to running Open Match.  [Here is a guide around security for Google Kubernetes Enginge on GCP](https://cloud.google.com/blog/products/gcp/exploring-container-security-running-a-tight-ship-with-kubernetes-engine-1-10), and a number of other guides are available from reputable sources on the internet.
### Minimum permissions on Kubernetes
* The components of Open Match should be run in a separate Kubernetes namespace if you're also using the cluster for other services. As of 0.3.0 they run in the 'default' namespace if you follow the development guide. 
* Note that the default MMForc process has cluster management permissions by default. Before moving to production, you should create a role with only access to create kubernetes jobs and configure the MMForc to use it.
### Kubernetes Jobs (MMFOrc)
The 0.3.0 MMFOrc component runs your MMFs as Kubernetes Jobs. You should periodically delete these jobs to keep the cluster running smoothly.  How often you need to delete them is dependant on  how many you are running.  There are a number of open source solutions to do this for you. ***Note that once you delete the job, you won't have access to that job's logs anymore unless you're sending your logs from kubernetes to a log aggregator like Google Stackdriver.  This can make it a challenge to troubleshoot issues***

## Open Match config
Debug logging and the extra debug code paths should be disabled in the `config/matchmaker_config.json` file (as of the time of this writing, 0.3.0).

## Public APIs for Open Match
In many cases, you may choose to configure your game clients to connect to the Open Match Frontend API, and in a few select cases (such as using it for P2P non-dedicated game server hosting), the game client may also need to connect to the Backend API.  In these cases, it is important to secure the API endpoints against common attacks, such as DDoS or malformed packet floods.  
* Using a cloud provider's Load Balancer in front of the Kubernetes Service is a common approach to enable vendor-specific DDoS protections.  Check the documentation for your cloud vendor's Load Balancer for more details ([GCP's DDoS protection](https://cloud.google.com/armor/)).
* Using an API framework can be used to limit endpoint access to only game clients you have authenticated using your platform's authentication service.  This may be accomplished with simple authentication tokens or a more complex scheme depending on your needs.

## Testing
(as of 0.3.0) The provided test programs are just for validating that Open Match is operating correctly; they are command-line applications designed to be run from within  the same cluster as Open Match and are therefore not a suitable test harness for doing production testing to make sure your matchmaker is ready to handle your live game.  Instead, it is recommended that you integrate Open Match into your game client and test it using the actual game flow players will use if at all possible.

### Load testing
Ideally, you would already be making 'headless' game clients for automated qa and load testing of your game servers; it is recommended that you also code these testing clients to be able to act as a mock player connecting to Open Match.  Load testing platform services is a huge topic and should reflect your actual game access patterns as closely as possible, which will be very game dependant. 
**Note: It is never a good idea to do load testing against a cloud vendor without informing them first!**
