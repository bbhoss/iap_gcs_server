# IAP-Protected GCS Bucket Web Server
This is a simple web server that serves files from a GCS bucket. The bucket is protected by IAP, so only users with the correct permissions can access the files.

## Setup
1. Enable Identity-Aware Proxy API on the project, including configuring the OAuth consent screen.
2. Create a GCS bucket and upload the files you want to serve.
3. Create a service account and grant `roles/storage.objectViewer` role on the bucket.
4. Rename app.yaml.example to app.yaml and fill in the missing values.
5. Deploy the app to App Engine `gcloud app deploy`
6. Enable IAP on the App Engine app.
7. Grant `roles/iap.httpsResourceAccessor` role on the App Engine app to the users you want to be able to access the files.

## Environment Variables

* `GCS_BUCKET` - The name of the GCS bucket to serve files from.
* `IAP_AUDIENCE` - The audience claim to verify when authenticating requests. See the example in app.yaml, or copy the value from the IAP settings page once enabled.

## Serving Behavior

The web server will serve any object it is allowed to read from the bucket configured with the `GCS_BUCKET` environment variable. The path of the request will be appended to the bucket name to determine the object to serve. For example, if the bucket is named `my-bucket` and the request is for `/foo/bar.txt`, the object `gs://my-bucket/foo/bar.txt` will be served. Additionally, if the object is not found, the server will attempt to serve `gs://my-bucket/foo/bar.txt/index.html`. This allows for serving static sites with clean URLs.