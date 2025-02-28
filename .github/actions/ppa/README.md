# PPA Builder

## Testing

To test, first build the registry image locally:

```sh
cd .github/actions/ppa/registry-image
docker build --platform linux/amd64 -t driskell/log-courier:ppa .
```

Then in the root of the repository, link up the `.main` folder, and then run the builder with the necessary environment arguments. You'll need to base64 encode an export of the GPG private key for signing and store it in the `$GNU_PG` environment variable (do not let it store in your shell history). Set `SKIP_SUBMIT` to `0` or remove it if you want to allow it to submit the results to PPA.

```sh
ln -nsf . .main
docker run --rm -v .:/github/workspace -w /github/workspace -e NAME=log-courier -e VERSION=v2.12.0 -e REF=v2.12.0 -e DRELEASE=1 -e SKIP_SUBMIT=1 -e GNU_PG="$GNU_PG" driskell/log-courier:ppa
```
