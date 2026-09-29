# Email verification gate

Unaffiliated and affiliated flows assume the account email is under the user's control. Without a hard gate, registration alone would let unverified sessions reach Onboarding, Affiliation actions, and the stream picker.

**Decision:** After auth, unverified users are limited to the **Verification waiting path**. Email verification is established by completing the Appwrite verification link, accepting an Invitation with its secret, or completing password recovery. At ship, grandfather existing users with a one-time Appwrite `Users.UpdateEmailVerification` backfill; the synthetic platform-admin from `ADMIN_EMAIL` bypasses the gate; seed identities used for local login must ship with `emailVerification` set so they are not stuck unverified. Appwrite sends verification mail (not the Attesta Mailer).

**Rejected:** Soft/nag-only verification; Attesta SMTP for verification messages; leaving seed login users unverified.
