# Mixin Virtual Machine

Based on the Mixin Trusted Group technology, it's possible to make a huge MTG network that allows developers to deploy smart contracts without building their own groups. This network can be run by many nodes with some kind of PoS consensus rules, thus it could provide better reputation than every small trusted group.

The network doesn't run any smart contract virtual machines, instead it needs to work with existing networks, e.g. Ethereum, EOS or Solana. Let's assume the smart contract network is Ethereum, the idea is pretty straitforward.

1. The group receives a transaction from some Mixin Messenger users.
2. The group should invoke the group contract on Ethereum, all according to some registered rules in the transaction extra field. 
3. Then the group contract could parse the message and store the Mixin transaction details in its storage.
4. Now any developer contracts can be notified that something has happened in the group contract, and they query it for recent transactions related to themselves.
6. In any case that some developer contract needs to send some message or money to some Mixin Messenger users, it should invoke the group contract with a message.
7. The group monitor all calls on the group contract and decide whether to send something to some Mixin Messenger users.

## Message

Mixin user to group memo.

Mixin group to contract extra.

Group contract data.

Developer contract to group contract extra.

## Performance

Multiple groups, multiple group contracts, multiple smart contract networks.

## Security

Group contract signature.

Developer contracts balance.

## Privacy

Masked user id for different contracts.
