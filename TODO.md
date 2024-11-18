# TODO

## Blockchain sync

This is new code and all needs writing.

 - simple peer seed/discovery/exchange (no TURN or NAT breaking)
 - sidecar data validation for chain with checkpoints
 - rollback to enable speculative updates? N blocks before trim?
 - what if we receive a conflict? (fail early!)


## UI pieces

These have functionality in the simulator, but no bespoke commands.


   - [authority] initial election setup and creation of the "chain" with the genesis block. This actually involves a multi-step process where the trustee and registrar entities must be involved in signing their portion of the setup data. The UI must allow for this "init-creation" and "continue-creation" for the genesis block.

   - [trustee] They have 4 roles:
    - keypair creation and signing of the initial params for the authority in the process of creating the genesis block
    - during parameter confirmation with encrypted shares for the other trustees
    - during parameter confirmation with the public key (confirming the shares)
    - producing a partial tally in the final section. (once enough partial tallys are created, we can show a result).

   - [registrar] Has 2 roles:
    - keypair creation and signing of the initial params for the authority in the process of creating the genesis block.
    - providing an "authentication" service on the registration URL (maybe a GtHub auth based solution, but local registrar with user/password file (htaccess?) would be simpler...) to sign voter registration requests

   - [voter] needs to:
     - create a keypair, register with a registrar, which involves visiting a URL in a browser, and creating the user registration block.
     - cast a vote

   - [auditor] verify (post-facto, live) and show tally. (post-facto is done, we have a full offline audit process, live is not done)


# notes

to vote/register we don't want the full chain.
we want "fetch" last block from a good node?
create our block, submit to chain (and repeat on failure)

The real missing piece is realtime block validation via recieved blocks from external peers and the associated speculative chain extension. This will involve "checkpointing" the election data at various stages and being able to rollback to a previous state when we want to validate a separate chain fragment. That in itself is a DoS vector forcing us to do work to validate invalid chains.

Also, how do we reach consensus on the final few blocks once the time is past and no new blocks can be added. perhaps we should allow a results block to be added by anyone and after N result blocks are added the result is considered fixed.

I haven't consider after-the-fact complete chain rewrites, which might be feasible... I guess we rely on publishing of the final block ID so the ElectionID + final BlockID make an immutable pair, published widely to cement confidence?

I think the requirement for an auditor node is essential after the genesis block is created. The auditor node will be responsible for accepting new blocks, including the ones we want to add to the chain locally (ie. as a result of a user action).

