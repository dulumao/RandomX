# RandomX (Golang Implementation)
### Proposed Use and Analysis of RandomX on the DERO Network



https://medium.com/deroproject/analysis-of-randomx-dde9dfe9bbc6



RandomX POW Golang Implementation and Analysis by the DERO Team. We are looking for an ASIC-resistant POW algorithm, so we decided to evaluate RandomX.

Findings: 

The sum total of the protection depends on the CFROUND instruction, which basically defines rounding in different modes. This is a hardware dependent implementation. This means, if Intel/AMD/ARM or others change/fix their implementation for whatever reason, all RandomX blockchains implementations will encounter issues. This is too big a risk to undertake, almost handing over the control to others.

This is a Pure GO software implementation as Proof-OF-Concept. The test cases are same as the original RandomX implementation. All test cases passed except Test B, which is due to different round instruction. This is because the actual implementation of rounding is not documented by processor manufacturers. Also, note that the rounding functionality used is not used in 99.9999% of software. Thus, it can be removed/modified anytime.

NOTE: Rounding CFROUND instruction in vm_instruction.go line 822, still has a bug which showcases for Test B.

Based on above findings we have decided not to use the RandomX algorithm on the DERO Network to avoid any breakdown in future.

Please find attached RandomX Golang implementation. Code needs severe cleanup and formating.

NB: Above views are limited and personal of DERO Team.
