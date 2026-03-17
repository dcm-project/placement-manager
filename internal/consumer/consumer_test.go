package consumer_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/dcm-project/placement-manager/internal/consumer"
	"github.com/dcm-project/placement-manager/internal/store"
	"github.com/dcm-project/placement-manager/internal/store/model"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var _ = Describe("StatusConsumer", func() {
	var (
		db        *gorm.DB
		dataStore store.Store
		nc        *nats.Conn
		js        jetstream.JetStream
		sc        *consumer.StatusConsumer
		ctx       context.Context
		cancel    context.CancelFunc
		natsURL   string
		streamID  string
	)

	BeforeEach(func() {
		var err error

		// Setup in-memory database
		db, err = gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		Expect(err).NotTo(HaveOccurred())
		sqlDB, err := db.DB()
		Expect(err).NotTo(HaveOccurred())
		sqlDB.SetMaxOpenConns(1)
		Expect(db.AutoMigrate(&model.Resource{})).To(Succeed())
		dataStore = store.NewStore(db)

		// Use the NATS test server URL from suite_test.go
		natsURL = testNATSServer.ClientURL()

		// Connect a publisher client with JetStream
		nc, err = nats.Connect(natsURL)
		Expect(err).NotTo(HaveOccurred())
		js, err = jetstream.New(nc)
		Expect(err).NotTo(HaveOccurred())

		// Use unique stream/consumer names per test to avoid conflicts
		streamID = uuid.New().String()[:8]

		// Create and start the consumer
		sc, err = consumer.New(natsURL, "dcm.*", dataStore,
			consumer.WithStreamName("test-stream-"+streamID),
			consumer.WithConsumerName("test-consumer-"+streamID),
		)
		Expect(err).NotTo(HaveOccurred())

		ctx, cancel = context.WithCancel(context.Background())
		Expect(sc.Start(ctx)).To(Succeed())
	})

	AfterEach(func() {
		sc.Stop()
		// Delete the stream to free the subjects for the next test
		_ = js.DeleteStream(context.Background(), "test-stream-"+streamID)
		nc.Close()
		cancel()
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})

	publishStatusEvent := func(providerName, serviceType, instanceID, status, message string) {
		event := cloudevents.New()
		event.SetID(uuid.New().String())
		event.SetSource(fmt.Sprintf("dcm/providers/%s", providerName))
		event.SetType(fmt.Sprintf("dcm.status.%s", serviceType))
		event.SetTime(time.Now())

		payload := consumer.VMEvent{
			Id:        instanceID,
			Status:    status,
			Message:   message,
			Timestamp: time.Now(),
		}
		Expect(event.SetData(cloudevents.ApplicationJSON, payload)).To(Succeed())

		data, err := json.Marshal(event)
		Expect(err).NotTo(HaveOccurred())

		subject := fmt.Sprintf("dcm.%s", serviceType)
		_, err = js.Publish(ctx, subject, data)
		Expect(err).NotTo(HaveOccurred())
	}

	createResource := func(instanceID string) {
		provider := "test-provider"
		approval := "APPROVED"
		r := model.Resource{
			ID:                    uuid.New().String(),
			CatalogItemInstanceId: instanceID,
			Spec:                  map[string]any{"cpu": "2"},
			ProviderName:          &provider,
			ApprovalStatus:        &approval,
			Path:                  "resources/" + uuid.New().String(),
		}
		_, err := dataStore.Resource().Create(ctx, r)
		Expect(err).NotTo(HaveOccurred())
	}

	It("updates resource status on valid VM status event", func() {
		instanceID := uuid.New().String()
		createResource(instanceID)

		publishStatusEvent("kubevirt-sp", "vm", instanceID, "RUNNING", "VM is running")

		Eventually(func() string {
			var res model.Resource
			db.Where("catalog_item_instance_id = ?", instanceID).First(&res)
			return res.Status
		}, 2*time.Second, 100*time.Millisecond).Should(Equal("RUNNING"))
	})

	It("updates resource status on valid container status event", func() {
		instanceID := uuid.New().String()
		createResource(instanceID)

		publishStatusEvent("podman-sp", "container", instanceID, "SUCCEEDED", "Container completed")

		Eventually(func() string {
			var res model.Resource
			db.Where("catalog_item_instance_id = ?", instanceID).First(&res)
			return res.Status
		}, 2*time.Second, 100*time.Millisecond).Should(Equal("SUCCEEDED"))
	})

	It("updates resource status on valid cluster status event", func() {
		instanceID := uuid.New().String()
		createResource(instanceID)

		publishStatusEvent("k8s-sp", "cluster", instanceID, "ACTIVE", "Cluster is active")

		Eventually(func() string {
			var res model.Resource
			db.Where("catalog_item_instance_id = ?", instanceID).First(&res)
			return res.Status
		}, 2*time.Second, 100*time.Millisecond).Should(Equal("ACTIVE"))
	})

	It("updates status message along with status", func() {
		instanceID := uuid.New().String()
		createResource(instanceID)

		publishStatusEvent("kubevirt-sp", "vm", instanceID, "FAILED", "VM crashed unexpectedly")

		Eventually(func() string {
			var res model.Resource
			db.Where("catalog_item_instance_id = ?", instanceID).First(&res)
			return res.StatusMessage
		}, 2*time.Second, 100*time.Millisecond).Should(Equal("VM crashed unexpectedly"))
	})

	It("discards events with invalid status for service type", func() {
		instanceID := uuid.New().String()
		createResource(instanceID)

		// "ACTIVE" is not a valid VM status
		publishStatusEvent("kubevirt-sp", "vm", instanceID, "ACTIVE", "invalid")

		Consistently(func() string {
			var res model.Resource
			db.Where("catalog_item_instance_id = ?", instanceID).First(&res)
			return res.Status
		}, 500*time.Millisecond, 100*time.Millisecond).Should(BeEmpty())
	})

	It("discards events with unknown service type", func() {
		instanceID := uuid.New().String()
		createResource(instanceID)

		// dcm.unknown matches dcm.* so it goes through JetStream but is rejected by handler
		event := cloudevents.New()
		event.SetID(uuid.New().String())
		event.SetSource("dcm/providers/unknown-sp")
		event.SetType("dcm.status.unknown")
		event.SetTime(time.Now())
		payload := consumer.VMEvent{
			Id:        instanceID,
			Status:    "RUNNING",
			Timestamp: time.Now(),
		}
		Expect(event.SetData(cloudevents.ApplicationJSON, payload)).To(Succeed())
		data, err := json.Marshal(event)
		Expect(err).NotTo(HaveOccurred())
		_, err = js.Publish(ctx, "dcm.unknown", data)
		Expect(err).NotTo(HaveOccurred())

		Consistently(func() string {
			var res model.Resource
			db.Where("catalog_item_instance_id = ?", instanceID).First(&res)
			return res.Status
		}, 500*time.Millisecond, 100*time.Millisecond).Should(BeEmpty())
	})

	It("handles events for non-existent instances gracefully", func() {
		publishStatusEvent("kubevirt-sp", "vm", "non-existent-id", "RUNNING", "VM is running")

		// Give it time to process - no panic expected
		time.Sleep(200 * time.Millisecond)
	})

	It("discards malformed CloudEvent messages", func() {
		_, err := js.Publish(ctx, "dcm.vm", []byte("not-valid-json"))
		Expect(err).NotTo(HaveOccurred())

		// Give it time to process - no panic expected
		time.Sleep(200 * time.Millisecond)
	})

	It("handles multiple sequential status updates", func() {
		instanceID := uuid.New().String()
		createResource(instanceID)

		publishStatusEvent("kubevirt-sp", "vm", instanceID, "PROVISIONING", "Starting VM")

		Eventually(func() string {
			var res model.Resource
			db.Where("catalog_item_instance_id = ?", instanceID).First(&res)
			return res.Status
		}, 2*time.Second, 100*time.Millisecond).Should(Equal("PROVISIONING"))

		time.Sleep(200 * time.Millisecond)

		publishStatusEvent("kubevirt-sp", "vm", instanceID, "RUNNING", "VM is running")

		Eventually(func() string {
			var res model.Resource
			db.Where("catalog_item_instance_id = ?", instanceID).First(&res)
			return res.Status
		}, 2*time.Second, 100*time.Millisecond).Should(Equal("RUNNING"))
	})
})
