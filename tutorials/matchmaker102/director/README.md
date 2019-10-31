This folder provides a sample Director for Open Match Matchmaker 101 Tutorial.
<TODO - Update the README with details of the Director and the steps that need
to be run before executing the commamds below>

Run the below steps in this folder to set up the Director.

Step1: Specify your Registry URL.
```
REGISTRY=[YOUR_REGISTRY_URL]
```

Step2: Build the Director image.
```
docker build -t $REGISTRY/mm101-tutorial-director .
```

Step3: Push the Director image to the configured Registry.
```
docker push $REGISTRY/mm101-tutorial-director
```

Step4: Update the install yaml for your setup.
```
sed "s|REGISTRY_PLACEHOLDER|$REGISTRY|g" director.yaml | kubectl apply -f -
```
