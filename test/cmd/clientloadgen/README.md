# Client load generator

This is an application that dumps players into the pool by writing to Redis state storage. As it writes directly to Redis state storage to simulate clients hitting the frontend API, it does not generate any load on the frontend API (or, in fact, even need it to be running). 

Only to be used for testing, and only in isolated environments (not in production!)

# Ping files
This application requires files filled with statistics used to generate simulated client latencies.  Example files with placeholder data (with every possible route taking 75ms) are provided.  More realistic data can be found on public sites such as https://wondernetwork.com/wonderproxy or you can provide your own in the format below.

## Format
Ping files are tab-delimited data, in this order:
`City	Distance	Average	% of SOLf/o	min	max	mdev	Last Checked`

## city.percent file
This is a file with of all the cities you want to generate simulated clients from, and associated numbers representing the percent of clients to simulate from those cities. All the percentages are normalized to fall within the range of 0 - 1 (floating point). Developers who want to modify the file should consult the source code to learn more of how it works and is used. 

# Building
Easiest using Google Cloud Build:
```
gcloud builds submit --substitutions TAG_NAME=latest --config cloudbuild.yaml
```

Simple kubectl command to run it and see the results.  Replace REGISTRTY_URI with a docker image registry your k8s cluster can access.  If you're using cloudbuild it's probably `gcr.io/<PROJECT_NAME>`, and was output at the end of the Cloud Build log.
```
kubectl run --rm --restart=Never --image-pull-policy=Always -i --tty --image=<REGISTRY_URI>/openmatch-clientloadgen om-clientloadgen
```

# Further reading
More information about this program can be found in the [development guide](../../../docs/development.md).
