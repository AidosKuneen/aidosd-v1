# Incompatibilities

## Common Differences with Bitcon
* These APIs don't have full features, these are just for a few exchange programs.
* Error codes and error string from these commands are not same as ones from bitcoin.
* Deposit addresses must be changed per every deposits e.g. by calling `getnewaddress` on your exchange system by your own. This library doesn't care about the changing addresses.
* Formats of addresses, hashes, transactions etc are COMPLETELY different with ones in Bitcon.
* You cannot use "watch-only address" and "transaction comment" functions. All of these parameters are ignored.
* Confirmations in ADK are regarded as "finalized", so all parameter for
 number of comfirmations  are ignored.

## `getnewaddress`

| Parameter        | Incompatibility Note  |
| ------------- |------------- |
| Account      | --- | 

| Result   | Incompatibility Note  |
| ------------- |------------- |
| result      | ---| 


## `listaccounts`

| Parameter        | Incompatibility Note  |
| ------------- |------------- |
| Confirmations      | ignored| 
| Include Watch-Only      | ignored| 

| Result   | Incompatibility Note  |
| ------------- |------------- |
| result      | ---| 
| →Account : Balance     | ---| 

## `listaddressgroupings`

| Parameter        | Incompatibility Note  |
| ------------- |------------- |
|       | | 


| Result   | Incompatibility Note  |
| ------------- |------------- |
| result      | ---|
| →Groupings      | always 1 group|
|→ →Address Details     | ---|
| → → →Address      | ---|
| → → →Balance      | ---|
| → → →Account      | ---|

## `validateaddress`

| Parameter        | Incompatibility Note  |
| ------------- |------------- |
| Address      | ---| 

| Result   | Incompatibility Note  |
| ------------- |------------- |
| result      | ---|
| →isvalid       | ---|
| →address      | ---|
| →scriptPubKey       | always empty string|
| →ismine       | ---|
|  →iswatchonly      |exists and false if address is in the wallet|
| →isscript      | exists and false  if address is in the wallet|
| →script       | always empty string|
| →hex       | always doesn't exist|
|  →addresses      |always doesn't exist|
|  →→Address     | always doesn't exist|
|  →sigrequired      | always doesn't exist|
|  →pubkey     | exists and empty string  if address is in the wallet|
|  →iscompressed      | exists and false if address is in the wallet|
|  →account      | ---|
|  →hdkeypath     | always doesn't exist|
|   →hdmasterkeyid 	    |always doesn't exist|


##  `settxfee`

| Parameter        | Incompatibility Note  |
| ------------- |------------- |
| Transaction Fee      | Ignored | 

| Result   | Incompatibility Note  |
| ------------- |------------- |
| result      | always true| 

##  `walletpassphrase`

| Parameter        | Incompatibility Note  |
| ------------- |------------- |
| Passphrase      | ---| 
| Seconds      | ---| 

| Result   | Incompatibility Note  |
| ------------- |------------- |
| result      | ---| 

##  `sendmany`

| Parameter        | Incompatibility Note  |
| ------------- |------------- |
| From Account      | ---| 
|Outputs      | ---| 
|→Address/Amount   | ---| 
| Confirmations      | ignored |
|Comment     |ignored |
|Subtract Fee From Amount     | ignored |
|→Address     |ignored |

| Result   | Incompatibility Note  |
| ------------- |------------- |
| result      | ---| 

##  `sendfrom`

| Parameter        | Incompatibility Note  |
| ------------- |------------- |
| From Account      | ---| 
|To Address      | ---| 
|Amount   | ---| 
| Confirmations      | ignored |
|Comment     |ignored |
|Comment To    |ignored |

| Result   | Incompatibility Note  |
| ------------- |------------- |
| result      | None| 

##  `gettransaction`

| Parameter        | Incompatibility Note  |
| ------------- |------------- |
| TXID      | bundle hash| 
| Include Watch-Only      | ignored| 

| Result   | Incompatibility Note  |
| ------------- |------------- |
| result      | ---| 

##  `getbalance`

| Parameter        | Incompatibility Note  |
| ------------- |------------- |
| Account      | ---| 
| Confirmations      | ignored |
| Include Watch-Only      | ignored| 

| Result   | Incompatibility Note  |
| ------------- |------------- |
| result      | ---| 

##  `sendtoaddress`

| Parameter        | Incompatibility Note  |
| ------------- |------------- |
|To Address      | ---| 
|Amount   | ---| 
| Confirmations      | ignored |
|Comment     |ignored |
|Comment To    |ignored |
|Subtract Fee From Amount     | ignored |

| Result   | Incompatibility Note  |
| ------------- |------------- |
| result      | ---| 

##  `listtransactions`

| Parameter        | Incompatibility Note  |
| ------------- |------------- |
| Account      | ---| 
| Count      | ---| 
| Skip      | ---| 
| Include Watch-Only      | ignored| 

| Result   | Incompatibility Note  |
| ------------- |------------- |
| result      | None|  
