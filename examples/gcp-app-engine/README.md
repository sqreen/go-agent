# Google App Engine Example

This example allows to create a Go web server that serves "Hello HTTP!".
It firstly uses Google Cloud Build with a multi-stage dockerfile in order to build the
docker image containing the Go server. It is then deployed to Google App
Engine using the flexible environment.

NOTE: The commands below assume that you have set `$PROJECT_ID` to the name of
your GCP project. If your project name is `my-project`, you can do this by
executing `export PROJECT_ID=my-project` prior to running the below commands.

1. Create the docker image with Cloud Build:
   ```console
   $ gcloud builds submit --config=cloudbuild.yaml .
   ```

1. Get your Sqreen credentials from our dashboard at https://my.sqreen.com/new-application#golang-agent

1. Deploy the previously built docker image to Google App Engine flexible
   environment:
   ```console
   $ gcloud app deploy --image-url=gcr.io/$PROJECT_ID/sqreen-go-hello-http app.yaml
   ```

You can then hit the URL returned by that command to see the `Hello HTTP!` web
service response, now protected by Sqreen!

Don't forget to tear down your App Engine deployment to avoid billing charges
for a running service.
