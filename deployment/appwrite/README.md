# Appwrite Deployment Module

This directory vendors the official Appwrite self-hosting baseline used by Attesta.

Upstream source
- Installation docs: `https://appwrite.io/docs/advanced/self-hosting/installation`
- Compose template: `https://appwrite.io/install/compose`
- Environment template: `https://appwrite.io/install/env`
- Pinned Appwrite image in the vendored compose: `appwrite/appwrite:1.8.1`

Files
- `docker-compose.appwrite.yaml`: upstream Appwrite compose baseline, kept close to the official template.
- `.env.appwrite.example`: upstream Appwrite environment template, kept close to the official template.

Validation
```bash
docker compose \
  --env-file deployment/appwrite/.env.appwrite.example \
  -f deployment/appwrite/docker-compose.appwrite.yaml \
  config
```

Operator smoke checklist
1. Copy `.env.appwrite.example` to a local `.env` file for the target environment and replace the placeholder secrets.
2. Start the stack with `docker compose --env-file .env -f docker-compose.appwrite.yaml up -d --remove-orphans`.
3. Open the Appwrite console on the configured hostname and create the first console account.
4. Create the Attesta Appwrite project and record its project ID.
5. Create an API key for Attesta with the users, teams, memberships, labels, and storage scopes required by the identity adapter.
6. Create the org assets storage bucket and set the bucket ID that Attesta will use for organization logos.
7. Confirm invite and recovery URLs resolve back to Attesta before testing signup, invite, or reset flows.

Notes
- The vendored compose publishes ports `80` and `443` and creates the upstream `gateway` and `runtimes` Docker networks.
- Attesta-specific compose wrappers should add only the minimum wiring needed to connect Attesta to the Appwrite API; they should not fork the Appwrite service graph here.
