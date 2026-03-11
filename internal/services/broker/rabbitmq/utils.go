package rabbitmq

import amqp "github.com/rabbitmq/amqp091-go"

const (
	TasksQueue         = "tasks"
	TasksProgressQueue = "tasks_progress"
	DeadLetterQueue    = "dead_letter"
)

func DefineQueues(conn *amqp.Connection, requeueLimit int) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}

	err = DefineTasksQueue(ch, requeueLimit)
	if err != nil {
		return err
	}

	err = DefineDeadLetterQueue(ch)
	if err != nil {
		return err
	}

	err = DefineTasksProgressQueue(ch)
	if err != nil {
		return err
	}
	return nil
}

func DefineTasksQueue(ch *amqp.Channel, deliveryLimit int) error {
	_, err := ch.QueueDeclare(
		TasksQueue,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-queue-type":              "quorum",
			"x-delivery-limit":          deliveryLimit,
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": DeadLetterQueue,
			"x-dead-letter-strategy":    "at-least-once",
		},
	)
	return err
}

func DefineDeadLetterQueue(ch *amqp.Channel) error {
	_, err := ch.QueueDeclare(
		DeadLetterQueue,
		true,
		false,
		false,
		false,
		nil,
	)
	return err
}

func DefineTasksProgressQueue(ch *amqp.Channel) error {
	_, err := ch.QueueDeclare(
		TasksProgressQueue,
		true,
		false,
		false,
		false,
		nil,
	)
	return err
}
