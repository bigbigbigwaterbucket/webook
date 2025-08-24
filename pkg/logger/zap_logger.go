package logger

import "go.uber.org/zap"

type ZapLogger struct {
	logger *zap.Logger
}

func (z *ZapLogger) Info(msg string, args ...Filed) {
	z.logger.Info(msg, z.toZapFiled(args)...)
}

func (z *ZapLogger) Debug(msg string, args ...Filed) {
	z.logger.Debug(msg, z.toZapFiled(args)...)
}

func (z *ZapLogger) Warn(msg string, args ...Filed) {
	z.logger.Warn(msg, z.toZapFiled(args)...)
}

func (z *ZapLogger) Error(msg string, args ...Filed) {
	z.logger.Error(msg, z.toZapFiled(args)...)
}

func (z *ZapLogger) toZapFiled(fileds []Filed) (zfiled []zap.Field) {
	for _, filed := range fileds {
		zfiled = append(zfiled, zap.Any(filed.Key, filed.Value))
	}
	return
}
