This folder provides a sample Match Function for Open Match backfill Tutorial.
<TODO - Update the README with details of the Match Function and the steps that need
to be run before executing the commamds below>

Run the below steps in this folder to set up the Match Function.

Step1: Specify your Registry URL.
```
REGISTRY=[YOUR_REGISTRY_URL]
```

Step2: Build the Match Function image.
```
docker build -t $REGISTRY/backfill-matchfunction .
```

Step3: Push the Match Function image to the configured Registry.
```
docker push $REGISTRY/backfill-matchfunction
```

Step4: Update the install yaml for your setup.
```
sed "s|REGISTRY_PLACEHOLDER|$REGISTRY|g" matchfunction.yaml | kubectl apply -f -
```
