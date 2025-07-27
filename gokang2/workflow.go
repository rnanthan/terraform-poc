package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// Data Transfer Objects
type OrderRequest struct {
	OrderID    string  `json:"order_id"`
	CustomerID string  `json:"customer_id"`
	Amount     float64 `json:"amount"`
	ProductID  string  `json:"product_id"`
}

type PaymentRequest struct {
	OrderID    string  `json:"order_id"`
	CustomerID string  `json:"customer_id"`
	Amount     float64 `json:"amount"`
}

type ShippingRequest struct {
	OrderID    string `json:"order_id"`
	CustomerID string `json:"customer_id"`
	ProductID  string `json:"product_id"`
	Address    string `json:"address"`
}

// Activity Interfaces and Implementations

// Order Activities
func ValidateOrder(ctx context.Context, request OrderRequest) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Validating order", "order_id", request.OrderID)

	if request.Amount <= 0 {
		return "", fmt.Errorf("invalid order amount: %f", request.Amount)
	}

	// Simulate validation time
	time.Sleep(100 * time.Millisecond)
	return "VALID", nil
}

func GetCustomerAddress(ctx context.Context, customerID string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Fetching address for customer", "customer_id", customerID)

	// Simulate database lookup
	time.Sleep(50 * time.Millisecond)
	return "123 Main St, City, State 12345", nil
}

func SendOrderConfirmation(ctx context.Context, orderID, customerID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending confirmation", "order_id", orderID, "customer_id", customerID)

	// Simulate sending email/notification
	time.Sleep(200 * time.Millisecond)
	return nil
}

func UpdateOrderStatus(ctx context.Context, orderID, status string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Updating order status", "order_id", orderID, "status", status)

	// Simulate database update
	time.Sleep(50 * time.Millisecond)
	return nil
}

// Payment Activities
func ChargeCustomer(ctx context.Context, customerID string, amount float64) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Processing payment", "customer_id", customerID, "amount", amount)

	// Simulate payment processing
	time.Sleep(500 * time.Millisecond)
	paymentID := fmt.Sprintf("PAY_%d", time.Now().Unix())
	return paymentID, nil
}

func RecordPayment(ctx context.Context, orderID, paymentID string, amount float64) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Recording payment", "order_id", orderID, "payment_id", paymentID)

	// Simulate database record
	time.Sleep(100 * time.Millisecond)
	return nil
}

// Shipping Activities
func CreateShipment(ctx context.Context, orderID, address string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Creating shipment", "order_id", orderID, "address", address)

	// Simulate shipment creation
	time.Sleep(300 * time.Millisecond)
	trackingNumber := fmt.Sprintf("TRACK_%d", time.Now().Unix())
	return trackingNumber, nil
}

func SchedulePickup(ctx context.Context, orderID, productID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Scheduling pickup", "order_id", orderID, "product_id", productID)

	// Simulate pickup scheduling
	time.Sleep(200 * time.Millisecond)
	return nil
}

func NotifyCustomer(ctx context.Context, customerID, trackingNumber string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Notifying customer", "customer_id", customerID, "tracking_number", trackingNumber)

	// Simulate notification
	time.Sleep(100 * time.Millisecond)
	return nil
}

// Child Workflow: Payment Processing
func PaymentWorkflow(ctx workflow.Context, request PaymentRequest) (string, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting payment workflow", "order_id", request.OrderID)

	// Activity options
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &workflow.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Charge customer
	var paymentID string
	err := workflow.ExecuteActivity(ctx, ChargeCustomer, request.CustomerID, request.Amount).Get(ctx, &paymentID)
	if err != nil {
		return "", fmt.Errorf("failed to charge customer: %w", err)
	}

	// Record payment
	err = workflow.ExecuteActivity(ctx, RecordPayment, request.OrderID, paymentID, request.Amount).Get(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to record payment: %w", err)
	}

	logger.Info("Payment workflow completed", "payment_id", paymentID)
	return paymentID, nil
}

// Child Workflow: Shipping Processing
func ShippingWorkflow(ctx workflow.Context, request ShippingRequest) (string, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting shipping workflow", "order_id", request.OrderID)

	// Activity options
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &workflow.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Create shipment
	var trackingNumber string
	err := workflow.ExecuteActivity(ctx, CreateShipment, request.OrderID, request.Address).Get(ctx, &trackingNumber)
	if err != nil {
		return "", fmt.Errorf("failed to create shipment: %w", err)
	}

	// Schedule pickup
	err = workflow.ExecuteActivity(ctx, SchedulePickup, request.OrderID, request.ProductID).Get(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to schedule pickup: %w", err)
	}

	// Notify customer
	err = workflow.ExecuteActivity(ctx, NotifyCustomer, request.CustomerID, trackingNumber).Get(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to notify customer: %w", err)
	}

	logger.Info("Shipping workflow completed", "tracking_number", trackingNumber)
	return trackingNumber, nil
}

// Parent Workflow: Order Processing
func OrderProcessingWorkflow(ctx workflow.Context, request OrderRequest) (string, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting order processing workflow", "order_id", request.OrderID)

	// Activity options for parent workflow
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &workflow.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Step 1: Update status to validating
	err := workflow.ExecuteActivity(ctx, UpdateOrderStatus, request.OrderID, "VALIDATING").Get(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to update order status: %w", err)
	}

	// Step 2: Validate the order
	var validationResult string
	err = workflow.ExecuteActivity(ctx, ValidateOrder, request).Get(ctx, &validationResult)
	if err != nil {
		workflow.ExecuteActivity(ctx, UpdateOrderStatus, request.OrderID, "INVALID").Get(ctx, nil)
		return "", fmt.Errorf("order validation failed: %w", err)
	}

	if validationResult != "VALID" {
		workflow.ExecuteActivity(ctx, UpdateOrderStatus, request.OrderID, "INVALID").Get(ctx, nil)
		return "Order validation failed", nil
	}

	// Step 3: Get customer address
	var customerAddress string
	err = workflow.ExecuteActivity(ctx, GetCustomerAddress, request.CustomerID).Get(ctx, &customerAddress)
	if err != nil {
		return "", fmt.Errorf("failed to get customer address: %w", err)
	}

	// Step 4: Update status to processing
	err = workflow.ExecuteActivity(ctx, UpdateOrderStatus, request.OrderID, "PROCESSING").Get(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to update order status: %w", err)
	}

	// Step 5: Execute child workflows in parallel
	childWorkflowOptions := workflow.ChildWorkflowOptions{
		WorkflowExecutionTimeout: 10 * time.Minute,
		WorkflowTaskTimeout:      time.Minute,
	}

	// Payment child workflow
	paymentWorkflowOptions := childWorkflowOptions
	paymentWorkflowOptions.WorkflowID = fmt.Sprintf("payment-%s", request.OrderID)
	paymentCtx := workflow.WithChildOptions(ctx, paymentWorkflowOptions)

	paymentFuture := workflow.ExecuteChildWorkflow(paymentCtx, PaymentWorkflow, PaymentRequest{
		OrderID:    request.OrderID,
		CustomerID: request.CustomerID,
		Amount:     request.Amount,
	})

	// Shipping child workflow
	shippingWorkflowOptions := childWorkflowOptions
	shippingWorkflowOptions.WorkflowID = fmt.Sprintf("shipping-%s", request.OrderID)
	shippingCtx := workflow.WithChildOptions(ctx, shippingWorkflowOptions)

	shippingFuture := workflow.ExecuteChildWorkflow(shippingCtx, ShippingWorkflow, ShippingRequest{
		OrderID:    request.OrderID,
		CustomerID: request.CustomerID,
		ProductID:  request.ProductID,
		Address:    customerAddress,
	})

	// Wait for both child workflows to complete
	var paymentID, trackingNumber string

	err = paymentFuture.Get(ctx, &paymentID)
	if err != nil {
		workflow.ExecuteActivity(ctx, UpdateOrderStatus, request.OrderID, "FAILED").Get(ctx, nil)
		return "", fmt.Errorf("payment workflow failed: %w", err)
	}

	err = shippingFuture.Get(ctx, &trackingNumber)
	if err != nil {
		workflow.ExecuteActivity(ctx, UpdateOrderStatus, request.OrderID, "FAILED").Get(ctx, nil)
		return "", fmt.Errorf("shipping workflow failed: %w", err)
	}

	// Step 6: Send confirmation
	err = workflow.ExecuteActivity(ctx, SendOrderConfirmation, request.OrderID, request.CustomerID).Get(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to send confirmation: %w", err)
	}

	// Step 7: Update final status
	err = workflow.ExecuteActivity(ctx, UpdateOrderStatus, request.OrderID, "COMPLETED").Get(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to update final status: %w", err)
	}

	result := fmt.Sprintf("Order processed successfully. Payment ID: %s, Tracking: %s", paymentID, trackingNumber)
	logger.Info("Order processing completed", "order_id", request.OrderID, "result", result)

	return result, nil
}

func main() {
	// Create temporal client
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort,
	})
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	// Create worker
	w := worker.New(c, "order-processing-queue", worker.Options{})

	// Register workflows
	w.RegisterWorkflow(OrderProcessingWorkflow)
	w.RegisterWorkflow(PaymentWorkflow)
	w.RegisterWorkflow(ShippingWorkflow)

	// Register activities
	w.RegisterActivity(ValidateOrder)
	w.RegisterActivity(GetCustomerAddress)
	w.RegisterActivity(SendOrderConfirmation)
	w.RegisterActivity(UpdateOrderStatus)
	w.RegisterActivity(ChargeCustomer)
	w.RegisterActivity(RecordPayment)
	w.RegisterActivity(CreateShipment)
	w.RegisterActivity(SchedulePickup)
	w.RegisterActivity(NotifyCustomer)

	// Start worker in goroutine
	go func() {
		err := w.Run(worker.InterruptCh())
		if err != nil {
			log.Fatalln("Unable to start worker", err)
		}
	}()

	// Give worker time to start
	time.Sleep(time.Second)

	// Execute workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        "order-12345",
		TaskQueue: "order-processing-queue",
	}

	request := OrderRequest{
		OrderID:    "ORDER-12345",
		CustomerID: "CUST-001",
		Amount:     99.99,
		ProductID:  "PROD-ABC",
	}

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, OrderProcessingWorkflow, request)
	if err != nil {
		log.Fatalln("Unable to execute workflow", err)
	}

	log.Println("Started workflow", "WorkflowID", we.GetID(), "RunID", we.GetRunID())

	// Get workflow result
	var result string
	err = we.Get(context.Background(), &result)
	if err != nil {
		log.Fatalln("Unable to get workflow result", err)
	}

	log.Println("Workflow result:", result)
}