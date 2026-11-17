# Email verification gate

Unaffiliated and affiliated flows assume the account email is under the user's control. Without a hard gate, registration alone would let unverified sessions reach Onboarding, Affiliation actions, and the stream picker.

**Decision:** After auth, unverified users are limited to the Verification waiting path. Email is verified by completing the provider verification link, accepting an Invitation with its secret, or completing password recovery. The synthetic platform-admin bypasses the gate; local seed login identities must ship already verified. Verification mail is sent by Appwrite, not the Attesta Mailer.

**Rejected:** Soft/nag-only verification; Attesta SMTP for verification messages; leaving seed login users unverified.
