# Vietnam payment policy research

Checked on 2026-10-21 for the proposed synthetic ecommerce gateway. This is a research baseline for scenario design, not a claim that the project implements a licensed Vietnamese payment service. The detailed authentication and dispute findings below were checked against the official texts of Circulars 50/2024, 77/2025, 18/2024 and 45/2025. The remaining sources identify related scope and follow-up work.

## What the reference knowledge contains

The sibling [card-fraud-platform knowledge folder](https://github.com/cvinhnguyen/card-fraud-platform/tree/35d4295a1e5a409fa242bc6aa9eff9b06038c111/generator/knowledge) contains five explicitly fictional coursework documents: card terms, authentication rules, dispute policy, fees and FAQ. Its [source loader](https://github.com/cvinhnguyen/card-fraud-platform/blob/35d4295a1e5a409fa242bc6aa9eff9b06038c111/generator/load_source.py) uploads these as bank documents for a support-agent retrieval corpus. They are not an executable legal rules engine or a bank's published policies.

Borrow their sectioned, citable document structure. Keep three distinct kinds of information:

- **Regulation:** official source, article/appendix, regulated role, transaction channel, effective dates and exceptions.
- **Provider policy:** a named bank, acquirer or card-network document and the product/version it governs.
- **Synthetic policy:** our explicitly fictional scenario assumptions, such as simulated insufficient funds, authentication failure or processor response delay.

Do not copy the fictional fee amounts, contactless PIN limit, OTP expiry/lockout counts, provisional-credit promises or liability statements into regulatory requirements.

## Official source register

| Source | Verified publication / effective information | Gateway relevance |
| --- | --- | --- |
| [Decree 52/2024/ND-CP](https://vanban.chinhphu.vn/?pageid=27160&docid=210262) | Issued 2024-05-15; effective 2024-07-01 | Non-cash payment framework; map the actual operator's role before interpreting payment-provider obligations. |
| [Circular 50/2024/TT-NHNN](https://vanban.chinhphu.vn/?pageid=27160&docid=211772) | Issued 2024-10-31; general effective date 2025-01-01, with staged provisions | Article 10 and Appendices 01–04 distinguish payment types and confirmation methods. |
| [Circular 77/2025/TT-NHNN](https://vanban.chinhphu.vn/?pageid=27160&docid=216580) | Issued 2025-12-31; general effective date 2026-03-01 | Article 10 replaces Appendices 01, 02 and 04 of Circular 50. Article 12 stages Articles 3 and 10 to 2026-07-01 for providers serving individuals and organizations, and 2026-10-01 for organization-only providers. |
| [Circular 18/2024/TT-NHNN, official gazette text](https://congbaocdn.chinhphu.vn/CongBaoCP/VanBan/2024/6/42190/50717-1-2024791-79218-2024-tt-nhnn.pdf) | Issued 2024-06-28 | Card operations; Articles 18–19 cover lost-card reports and disputes. |
| [Circular 45/2025/TT-NHNN](https://vanban.chinhphu.vn/?pageid=27160&docid=216016) | Issued 2025-11-19; effective 2026-01-05 | Amends Circular 18; Article 10 changes the regional reporting authority in Article 19(4), not the dispute timing provisions discussed below. |
| [Circular 41/2025/TT-NHNN](https://vanban.chinhphu.vn/?pageid=27160&docid=215823) and [22/2026/TT-NHNN](https://chinhphu.vn/?pageid=27160&docid=218213) | Effective 2025-11-05 and 2026-05-19 respectively | Both amend the payment-intermediary framework in Circular 40/2024; do not scope a real gateway using the 2024 text alone. Detailed operator obligations remain follow-up research. |
| [Personal Data Protection Law 91/2025/QH15](https://chinhphu.vn/?pageid=27160&docid=214590) | Issued 2025-06-26; effective 2026-01-01 | A real-data design needs a current privacy assessment, including implementing rules; relying only on a pre-2026 privacy checklist is insufficient. |

The register is a scoped starting point, not an exhaustive determination of amendments, licensing, AML, card-network rules or any bank's contractual obligations.

## Card checkout is different from a bank transfer

Circular 50 Article 10(1)(a) routes account/wallet payments and specified card transfers through Appendices 01–02. Article 10(1)(c) routes online card payments, excluding transfers, through Appendices 03–04. The reference's transfer example citing Decision 2345 must not become a universal card-checkout rule.

For the online-card category covered by Appendix 03, let `G` be the current transaction value and `T` the card's already-executed daily total at its issuer, with the exclusions stated in that appendix:

| Online card class | Appendix 03 classification | Confirmation direction |
| --- | --- | --- |
| E | `G + T <= 5,000,000 VND` | Appendix 04 permits password/PIN confirmation, including its prior-login condition. |
| F | `5,000,000 < G + T <= 100,000,000 VND` | Appendix 04 lists permitted methods, including OTP variants and other alternatives. |
| G | `G + T > 100,000,000 VND` | The replacement Appendix 04 lists advanced Soft OTP/Token OTP, FIDO, secure electronic signature, or EMV 3DS as alternatives. |

This is a summary of those categories, not a complete authentication algorithm: Article 10 contains other cases and exceptions, and stronger confirmation can cover lower classes. An EMV 3DS flow is not necessarily an SMS challenge. The fictional rule that every online card payment requires 3DS is broader than this source supports.

The gateway cannot infer the issuer-wide `T` from its own merchant traffic. In a simulation, an issuer fixture can supply that context. With a real provider, use its authoritative authentication result and agreed responsibilities. Do not pretend a gateway-only velocity feature is a regulatory daily total.

## Dispute timing is not a universal refund promise

Circular 18 Article 19 distinguishes the reporting window, investigation, compensation and unresolved cases:

- Article 19(1): the issuer's contractual window for a cardholder to request investigation/complaint must be **at least 60 days** from the transaction. This is not an automatic 60-day forfeiture rule.
- Article 19(2)(d): for cards with an SBV-issued BIN, the agreed handling period is at most **30 working days** from first receipt. For international-card-organization BINs, that period is agreed in the card contract.
- Article 19(3)(a): qualifying compensation is due within at most **5 working days after notification of the investigation result**, subject to the stated agreement, law and fault/force-majeure conditions. This does not establish the reference's blanket provisional refund within five days of reporting.
- Article 19(3)(b): unresolved responsibility after the agreed handling period has a further **15-working-day** provision for agreeing a handling approach or temporary compensation pending a final competent-authority conclusion.

Keep disputes, merchant refunds, processor uncertainty and fraud ground truth distinct. A complaint or authentication success is not itself a final finding about fraud or liability. Actual chargeback stages and network deadlines need the selected scheme/provider rules; the reference's 45/90-day values are fictional policy, not a universal legal timer.

## Mapping the research into the gateway

The [processor simulator](../../openspec/changes/build-minimum-payment-gateway/design.md#processor-simulator-store-and-faults) stands in for the external payment provider. It accepts a stable operation key, stores the simulated outcome and answers status queries. For example, it can persist success but delay the response past Payments' deadline: Payments records an unknown outcome, then recovers the existing success without charging twice. Fraud approval only permits proceeding to processing; it does not prove that the provider accepted or settled a payment.

| Concern | Proposed owner and behavior |
| --- | --- |
| Fraud risk | Fraud service applies versioned policy/model logic. Payments records its decision. A low-risk result does not bypass issuer authentication or guarantee processor success. |
| Issuer authentication | A later provider/issuer simulation models confirmation success, failure, expiry and cancellation. Bind the result to the attempt/operation and verify it at the provider boundary; never trust a browser's claimed success. |
| Financial outcome | Payments commits effect intents before provider calls, handles unknown outcomes and recovers with the same operation key. The simulator owns its operation database. |
| Vietnam currency scenarios | Introduce an explicit VND profile only with coherent contract, validation, amount-unit, generator and policy changes. The current fraud input accepts USD only; VND thresholds must not be pasted onto USD minor units. |
| Regulatory totals | Keep issuer-wide context with the issuer fixture. Share only permitted decision-time evidence with Fraud; hidden scenario truth must not leak into features. |
| Disputes and investigation | Preserve evidence and versioned policy citations. Add a separate lifecycle when implemented; do not overload processor timeout recovery or rewrite payment history. |
| Analyst knowledge | A later read-only investigation agent retrieves scoped, cited policy and case evidence. It cannot approve payments, grant refunds or change deployed rules from retrieved prose. |

The current `fraud.v2` contract has only `APPROVE` and `DECLINE`; it has no challenge workflow. Authentication-required, pending, expiry and cancellation behavior would need an explicit provider/checkout contract and lifecycle design. This research does not add those states to runtime or silently reinterpret existing events.

## Knowledge corpus and next research

Start with this source-backed note. If a retrieval corpus is implemented later, every document/section should carry a stable ID, language, source URL, issuing authority or fictional owner, document number, article/appendix, effective-from/to dates, amendment links, checked date and content hash. Preserve old versions for historical explanations and filter retrieval by transaction date, role and channel. Keep statutory text, provider terms and synthetic examples distinguishable in citations and answers.

Before selecting production policies, resolve the operator's legal role and licensing, the chosen bank/acquirer/network contracts, applicable AML and merchant-onboarding duties, privacy implementing rules and data-transfer/retention requirements. For the synthetic baseline, prioritize operation-key recovery first; add VND and simulated authentication as explicit later increments. No real biometrics, citizen-ID images, PAN, PIN or OTP values are needed for these fixtures.
