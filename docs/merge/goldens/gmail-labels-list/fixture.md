# gmail-labels-list fixtures

These goldens must be captured from the **same Google account** and the **same label set** for both `gws` and `native` providers. Otherwise parity will report breaking diffs (different label IDs) and the list-success case cannot be gated.

If CI shows many breaking diffs here, check that both provider fixtures come from the same account and comparable mailbox state.
