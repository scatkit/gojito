## JITO

* Relayer (TPU) is a smart transaction manager that forwards `interesting` non-vote transacion to Block Engine

* Block Engine:
    1. Accepts bundles submited from tranders
    2. simulates the bundles to selects transactions from transaction pool based on: fees, importance, how they can be bundled for MEV
    3. Winning bundles are forwarded to Validators
    
* Validator:
    1. Executes bundles automically & builds the blocks
    2. 
    

