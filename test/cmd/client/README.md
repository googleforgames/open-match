# Client load generator

This is an application that dumps players into the pool by writing to Redis state storage. As it writes directly to Redis state storage to simulate clients hitting the frontend API, it does not generate any load on the frontend API (or, in fact, even need it to be running). 

Only to be used for testing, and only in isolated environments (not in production!)

# Ping data
This application requires ping data in the same format as you will find in the `example/frontend/` directory.

# Building
Easiest using Google Cloud Build:
```
gcloud builds submit --substitutions TAG_NAME=latest --config cloudbuild.yaml
```

Simple kubectl command to run it and see the results.  Replace REGISTRTY_URI with a docker image registry your k8s cluster can access.  If you're using cloudbuild it's probably `gcr.io/<PROJECT_NAME>`, and was output at the end of the Cloud Build log.
```
kubectl run --rm --restart=Never --image-pull-policy=Always -i --tty --image=<REGISTRY_URI>/openmatch-clientloadgen om-clientloadgen
```
