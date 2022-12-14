package traceLib

import (
	"errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"net/http"
	"os"
)

/*
InitTracerProvider initializes tracerProvider and returns it. It also returns error which may happen during init process
Here is how to set it up during your server init(don't forget to add ExtractTraceInfoMiddleware in your router):

	tp, err := traceLib.InitTracerProvider("some name")
	tracer := tp.Tracer("some name")
	if err != nil {
		log.Println(err)
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()
*/
func InitTracerProvider(serviceName string) (*sdktrace.TracerProvider, error) {
	exp, err := newExporter()
	if err != nil {
		return nil, err
	}
	tp := newTraceProvider(exp, serviceName)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return tp, nil
}

/*
newExporter initializes jaeger.Exporter and returns it. It also returns error if JAEGER_ADDRESS not found or eager cant init exporter
*/
func newExporter() (*jaeger.Exporter, error) {
	addr := os.Getenv("JAEGER_ADDRESS")
	if addr == "" {
		return nil, errors.New("couldn't read .env variables for JAEGER_ADDRESS. Please check if you provided it correctly")
	}
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(addr)))
	if err != nil {
		return nil, err
	}
	return exp, nil
}

/*
newExporter initializes sdktrace.TracerProvider and returns it.
*/
func newTraceProvider(exp sdktrace.SpanExporter, serviceName string) *sdktrace.TracerProvider {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		),
	)

	if err != nil {
		panic(err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	)
}

/*
ExtractTraceInfoMiddleware is middleman function.
This middleware is intended to be used with an HTTP server and will extract trace information from the incoming request and attach it to the request's context.
This trace information can then be used downstream by other parts of the code to do things like log tracing information for requests.
*/
func ExtractTraceInfoMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
