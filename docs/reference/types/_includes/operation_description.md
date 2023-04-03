Operations are stateful external actions that FireFly triggers via plugins. They can succeed or fail.
They are grouped into Transactions in order to accomplish a single logical task.

The diagram below shows the different types of operation that are performed by each
FireFly plugin type. The color coding (and numbers) map those different types of operation
to the [Transaction](./transaction.html) types that include those operations.

[![FireFly operations by transaction type](../../images/operations_by_transaction_type.jpg)](../../images/operations_by_transaction_type.jpg)

### Operation status

When initially created an operation is in `Initialized` state. Once the operation has been successfully sent to its respective plugin to be processed its
status moves to `Pending` state. This indicates that the plugin is processing the operation. The operation will then move to `Succeeded` or `Failed`
state depending on the outcome.

In the event that an operation could not be submitted to the plugin for processing, for example because the plugin's microservice was temporarily
unavailable, the operation will remain in `Initialized` state. Re-submitting the same FireFly API call using the same idempotency key will cause FireFly
to re-submit the operation to its plugin.
