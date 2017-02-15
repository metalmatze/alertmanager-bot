package bot

//func (b *AlertmanagerBot) auth(message telebot.Message) error {
//	if message.Sender.ID != b.Config.TelegramAdmin {
//		commandsCounter.WithLabelValues("dropped").Inc()
//		return fmt.Errorf("unauthorized")
//	}
//
//	return nil
//}
//
//func (b *AlertmanagerBot) instrument(message telebot.Message) error {
//	command := message.Text
//	if _, ok := b.commands[command]; ok {
//		commandsCounter.WithLabelValues(command).Inc()
//		return nil
//	}
//
//	commandsCounter.WithLabelValues("incomprehensible").Inc()
//	return b.telegram.SendMessage(
//		message.Chat,
//		"Sorry, I don't understand...",
//		nil,
//	)
//}
