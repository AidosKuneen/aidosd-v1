# Incompatibilities

## Common Differences with Bitcon
* These APIs don't have full features, these are just for a few exchange programs.
* Error codes and error string from these commands are not same as ones from bitcoin.
* Deposit addresses must be changed per every deposits e.g. by calling `getnewaddress` on your exchange system by your own. This library doesn't care about the changing addresses.
* Formats of addresses, hashes, transactions etc are COMPLETELY different with ones in Bitcoin.
* You cannot use "watch-only address" and "transaction comment". All of these parameters are ignored.
* Confirmations in ADK are regarded as "finalized", so all parameters for
 number of comfirmations  are ignored.

## Details for Each APIs


### `getnewaddress`

| Parameter        | Incompatibility Note  |
| ------------- |------------- |
| Account      | --- | 

| Result   | Incompatibility Note  |
| ------------- |------------- |
| result      | ---| 


### `listaccounts`

| Parameter        | Incompatibility Note  |
| ------------- |------------- |
| Confirmations      | ignored| 
| Include Watch-Only      | ignored| 

| Result   | Incompatibility Note  |
| ------------- |------------- |
| result      | ---| 
| →Account : Balance     | ---| 

### `listaddressgroupings`

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

### `validateaddress`

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


###  `settxfee`

| Parameter        | Incompatibility Note  |
| ------------- |------------- |
| Transaction Fee      | Ignored | 

| Result   | Incompatibility Note  |
| ------------- |------------- |
| result      | always true| 

###  `walletpassphrase`

| Parameter        | Incompatibility Note  |
| ------------- |------------- |
| Passphrase      | ---| 
| Seconds      | ---| 

| Result   | Incompatibility Note  |
| ------------- |------------- |
| result      | ---| 

###  `sendmany`

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

###  `sendfrom`

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

###  `gettransaction`

| Parameter        | Incompatibility Note  |
| ------------- |------------- |
| TXID      | bundle hash| 
| Include Watch-Only      | ignored| 


| Result   | Incompatibility Note  |
| ------------- |------------- |
| result      | ---|  
| →amount 	     | ---|  
| →fee       | always 0|  
|→confirmations       | 0 if not confirmed, 100000 if confirmed|  
| →generated       |always doesn't exist|  
| →blockhash      | exists and empty string if confirmed|  
| →blockindex       |exists and 0 if confirmed|  
| →blocktime      | exists and same as the transaction timestamp if confirmed| 
|→txid        | bundle hash|  
| →walletconflicts       | always empty array|  
|  → →TXID      | always doesn't exist|  
|  →time      | same as transaction timestamp|  
| →timereceived       | same as transaction timestamp|  
| →bip125-replaceable      | always "no"|  
|  →comment      | always doesn't exists|
| →to      | always doesn't exists|
| →details      | ---|
| → →involvesWatchonly       | always doesn't exists|  
| → →account       | ---|  
|  → →address      | ---|  
|  → →category      | always "send" or "receive" |  
| →→amount 	     | ---|  
| → →vout       | always 0|  
| →→fee       | always 0|  
| → →abandoned       | exists and false if category is "send"|  
| → →hex       | always empty string|  

###  `getbalance`

| Parameter        | Incompatibility Note  |
| ------------- |------------- |
| Account      | ---| 
| Confirmations      | ignored |
| Include Watch-Only      | ignored| 

| Result   | Incompatibility Note  |
| ------------- |------------- |
| result      | ---| 

###  `sendtoaddress`

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

###  `listtransactions`

| Parameter        | Incompatibility Note  |
| ------------- |------------- |
| Account      | ---| 
| Count      | ---| 
| Skip      | ---| 
| Include Watch-Only      | ignored| 

| Result   | Incompatibility Note  |
| ------------- |------------- |
| result      | ---|  
| →Payment       | ---|  
| → →account       | ---|  
|  → →address      | ---|  
|  → →category      | always "send" or "receive" |  
|  → →amount 	     | can be 0|  
|  → →label     | always doesn't exist|  
| → →vout       | always 0|  
|→ →fee       | always 0|  
|→ →confirmations       | 0 if not confirmed, 100000 if confirmed|  
| → →trusted      | exists and false if not confirmed|  
| → →generated       |always doesn't exist|  
| → →blockhash      | exists and empty string if confirmed|  
| → →blockindex       |0 and empty string if confirmed|  
| → →blocktime      | exists and same as the transaction timestamp if confirmed|   
|→ →txid        | bundle hash|  
| → →walletconflicts       | always empty array|  
|  → → →TXID      | always doesn't exist|  
|  → →time      | same as  transaction timestamp|  
| → →timereceived       | same as  transaction timestamp|  
|  → →comment      | always doesn't exists)|
| → →to      | always doesn't exists|
| → →otheraccount      |always doesn't exists|
| → →bip125-replaceable      | always "no"|  
| → →abandoned       | exists and false if category is "send"|  
