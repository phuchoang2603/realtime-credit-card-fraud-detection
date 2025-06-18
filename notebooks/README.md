# Credit Card Fraud Detection Notebook

<!--toc:start-->

- [Credit Card Fraud Detection Notebook](#credit-card-fraud-detection-notebook)
  - [How to Run Jupyter Lab with Docker](#how-to-run-jupyter-lab-with-docker)
  - [Background](#background)
    - [Fraud Scenarios](#fraud-scenarios)
    - [Detection System Architecture](#detection-system-architecture)
    - [Machine Learning for Credit Card Fraud Detection](#machine-learning-for-credit-card-fraud-detection)
    - [Challenges in CCFD](#challenges-in-ccfd)
  - [Reference](#reference)
  <!--toc:end-->

## How to Run Jupyter Lab with Docker

To start the Jupyter Lab environment using Docker:

1. Open your terminal
2. Navigate to the `jupyter-lab-docker` directory:

   ```bash
   cd jupyter-lab-docker
   ```

3. Run the following command to build and start the container in the background:

   ```bash
   docker compose up --build -d
   ```

4. Open your browser and go to <http://localhost:8888> to access Jupyter Lab.
5. Use the token shown in the terminal logs to log in (or set up a password if configured).

## Background

### Fraud Scenarios

- **Card Present (CP):**

  - _Lost/Stolen Cards_: Legitimate card stolen or lost.
  - _Counterfeited Cards_: Card info skimmed and cloned — mitigated by EMV chip tech.
  - _Card Not Received_: Card intercepted before reaching the customer.
  - ✅ CP fraud trends are decreasing due to security measures.

- **Card Not Present (CNP):**

  - Conducted remotely: phone, email, internet.
  - Arises from leaked credentials (phishing, breaches).
  - Credentials often sold on underground marketplaces.
  - ❗ CNP fraud is harder to trace and more prevalent.

![Fig 1: Card Fraud Evolution](./experiment/images/card-fraud-evolution.png)

> **Fig. 1**: Card-not-present frauds dominate total fraud value across SEPA-issued cards.

---

### Detection System Architecture

![Fig 2: FDS Layers](./experiment/images/fds-layer.png)

> **Fig. 2**: Multiple layers of defense in a Fraud Detection System (FDS). This notebook focuses on the **data-driven model**.

1. **Terminal Control**

   - PIN verification, balance, card status, spending limits
   - Real-time queries to issuer server

2. **Transaction-Blocking Rules**

   - Handcrafted `if-else` rules (e.g., deny unsecured internet payments)
   - Fast, expert-defined, applied before feature aggregation

3. **Scoring Rules**

   - `if-else` logic on feature vectors
   - Assigns fraud scores based on patterns (e.g., continent mismatch)

4. **Data-Driven Model (DDM)**

   - ML classifier trained on labeled transaction data
   - Analyzes full feature vectors
   - Identifies fraud patterns beyond expert intuition

5. **Human Investigator**

   - Reviews alerts from scoring and ML
   - Provides labels (fraud/genuine) for feedback loops
   - High-risk alerts may bypass investigators (e.g., SMS confirmation)

---

### Machine Learning for Credit Card Fraud Detection

![Example Transactions Table](./experiment/images/sample-transaction.png)

- **Account-Related**: account ID, open date, limit, expiry
- **Transaction-Related**: timestamp, amount, merchant category, terminal info
- **Customer-Related**: customer ID, profile type

📌 Transactions labeled as:

- `TX_FRAUD = 0`: Genuine
- `TX_FRAUD = 1`: Fraudulent

![ML Pipeline](./experiment/images/baseline_ML_workflow_subset.png)

> **Fig. 3**: Common two-stage ML workflow: training on labeled data → predicting new transactions.

A prediction model $h(x, \theta)$ maps feature inputs $x$ to output label or risk score:

$$
L_{0/1}(y, \hat{y}) =
\begin{cases}
1 & \text{if } y \ne \hat{y} \\
0 & \text{if } y = \hat{y}
\end{cases}
$$

- Training: optimize parameters $\theta$ to minimize prediction loss
- Output: risk score or binary prediction
- Feature Engineering: transforms raw input into usable format

Common Algorithms

- **Logistic Regression (LR), Decision Trees (DT)** – simple, interpretable
- **Random Forests (RF), Boosting** – top-performing
- **Neural Networks (NN/DL)** – cutting-edge, used in research

---

### Challenges in CCFD

1. Class Imbalance

   - Fraud <1% of transactions
   - Need special techniques:
     - Sampling (over/under)
     - Weighted loss

2. Concept Drift

   - Changes in user/fraud behavior over time
   - Requires online/adaptive learning

3. Real-Time Constraints

   - Decisions must be made in milliseconds
   - Scalability is critical

4. Categorical Features

   - IDs, card types require transformation:
     - Aggregation
     - Embeddings
     - Graph features

5. Sequential Nature

   - Transaction histories per user or terminal
   - Model via time aggregation or sequence models (HMMs, RNNs)

6. Class Overlap

   - Fraud and genuine often look similar
   - Need strong contextual features

7. Metrics

   - Accuracy and AUC not enough
   - Balance **detection rate** vs **false positives**

8. Data Availability

   - Most datasets are private
   - Kaggle (2016) is only public benchmark
   - Lack of simulators hinders reproducibility

---

## Reference

Fraud Detection Handbook:
[https://fraud-detection-handbook.github.io/fraud-detection-handbook/Chapter_2_Background/CreditCardFraud.html](https://fraud-detection-handbook.github.io/fraud-detection-handbook/Chapter_2_Background/CreditCardFraud.html)
