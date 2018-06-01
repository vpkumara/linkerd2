// Code generated by protoc-gen-go. DO NOT EDIT.
// source: proxy/destination/destination.proto

/*
Package conduit_proxy_destination is a generated protocol buffer package.

It is generated from these files:
	proxy/destination/destination.proto

It has these top-level messages:
	Update
	AddrSet
	WeightedAddrSet
	WeightedAddr
	TlsIdentity
	NoEndpoints
*/
package conduit_proxy_destination

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import conduit_common "github.com/runconduit/conduit/controller/gen/common"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Update struct {
	// Types that are valid to be assigned to Update:
	//	*Update_Add
	//	*Update_Remove
	//	*Update_NoEndpoints
	Update isUpdate_Update `protobuf_oneof:"update"`
}

func (m *Update) Reset()                    { *m = Update{} }
func (m *Update) String() string            { return proto.CompactTextString(m) }
func (*Update) ProtoMessage()               {}
func (*Update) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type isUpdate_Update interface{ isUpdate_Update() }

type Update_Add struct {
	Add *WeightedAddrSet `protobuf:"bytes,1,opt,name=add,oneof"`
}
type Update_Remove struct {
	Remove *AddrSet `protobuf:"bytes,2,opt,name=remove,oneof"`
}
type Update_NoEndpoints struct {
	NoEndpoints *NoEndpoints `protobuf:"bytes,3,opt,name=no_endpoints,json=noEndpoints,oneof"`
}

func (*Update_Add) isUpdate_Update()         {}
func (*Update_Remove) isUpdate_Update()      {}
func (*Update_NoEndpoints) isUpdate_Update() {}

func (m *Update) GetUpdate() isUpdate_Update {
	if m != nil {
		return m.Update
	}
	return nil
}

func (m *Update) GetAdd() *WeightedAddrSet {
	if x, ok := m.GetUpdate().(*Update_Add); ok {
		return x.Add
	}
	return nil
}

func (m *Update) GetRemove() *AddrSet {
	if x, ok := m.GetUpdate().(*Update_Remove); ok {
		return x.Remove
	}
	return nil
}

func (m *Update) GetNoEndpoints() *NoEndpoints {
	if x, ok := m.GetUpdate().(*Update_NoEndpoints); ok {
		return x.NoEndpoints
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*Update) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _Update_OneofMarshaler, _Update_OneofUnmarshaler, _Update_OneofSizer, []interface{}{
		(*Update_Add)(nil),
		(*Update_Remove)(nil),
		(*Update_NoEndpoints)(nil),
	}
}

func _Update_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*Update)
	// update
	switch x := m.Update.(type) {
	case *Update_Add:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Add); err != nil {
			return err
		}
	case *Update_Remove:
		b.EncodeVarint(2<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Remove); err != nil {
			return err
		}
	case *Update_NoEndpoints:
		b.EncodeVarint(3<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.NoEndpoints); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("Update.Update has unexpected type %T", x)
	}
	return nil
}

func _Update_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*Update)
	switch tag {
	case 1: // update.add
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(WeightedAddrSet)
		err := b.DecodeMessage(msg)
		m.Update = &Update_Add{msg}
		return true, err
	case 2: // update.remove
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(AddrSet)
		err := b.DecodeMessage(msg)
		m.Update = &Update_Remove{msg}
		return true, err
	case 3: // update.no_endpoints
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(NoEndpoints)
		err := b.DecodeMessage(msg)
		m.Update = &Update_NoEndpoints{msg}
		return true, err
	default:
		return false, nil
	}
}

func _Update_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*Update)
	// update
	switch x := m.Update.(type) {
	case *Update_Add:
		s := proto.Size(x.Add)
		n += proto.SizeVarint(1<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case *Update_Remove:
		s := proto.Size(x.Remove)
		n += proto.SizeVarint(2<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case *Update_NoEndpoints:
		s := proto.Size(x.NoEndpoints)
		n += proto.SizeVarint(3<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

type AddrSet struct {
	Addrs []*conduit_common.TcpAddress `protobuf:"bytes,1,rep,name=addrs" json:"addrs,omitempty"`
}

func (m *AddrSet) Reset()                    { *m = AddrSet{} }
func (m *AddrSet) String() string            { return proto.CompactTextString(m) }
func (*AddrSet) ProtoMessage()               {}
func (*AddrSet) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *AddrSet) GetAddrs() []*conduit_common.TcpAddress {
	if m != nil {
		return m.Addrs
	}
	return nil
}

type WeightedAddrSet struct {
	Addrs        []*WeightedAddr   `protobuf:"bytes,1,rep,name=addrs" json:"addrs,omitempty"`
	MetricLabels map[string]string `protobuf:"bytes,2,rep,name=metric_labels,json=metricLabels" json:"metric_labels,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

func (m *WeightedAddrSet) Reset()                    { *m = WeightedAddrSet{} }
func (m *WeightedAddrSet) String() string            { return proto.CompactTextString(m) }
func (*WeightedAddrSet) ProtoMessage()               {}
func (*WeightedAddrSet) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *WeightedAddrSet) GetAddrs() []*WeightedAddr {
	if m != nil {
		return m.Addrs
	}
	return nil
}

func (m *WeightedAddrSet) GetMetricLabels() map[string]string {
	if m != nil {
		return m.MetricLabels
	}
	return nil
}

type WeightedAddr struct {
	Addr         *conduit_common.TcpAddress `protobuf:"bytes,1,opt,name=addr" json:"addr,omitempty"`
	Weight       uint32                     `protobuf:"varint,3,opt,name=weight" json:"weight,omitempty"`
	MetricLabels map[string]string          `protobuf:"bytes,4,rep,name=metric_labels,json=metricLabels" json:"metric_labels,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	TlsIdentity  *TlsIdentity               `protobuf:"bytes,5,opt,name=tls_identity,json=tlsIdentity" json:"tls_identity,omitempty"`
}

func (m *WeightedAddr) Reset()                    { *m = WeightedAddr{} }
func (m *WeightedAddr) String() string            { return proto.CompactTextString(m) }
func (*WeightedAddr) ProtoMessage()               {}
func (*WeightedAddr) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *WeightedAddr) GetAddr() *conduit_common.TcpAddress {
	if m != nil {
		return m.Addr
	}
	return nil
}

func (m *WeightedAddr) GetWeight() uint32 {
	if m != nil {
		return m.Weight
	}
	return 0
}

func (m *WeightedAddr) GetMetricLabels() map[string]string {
	if m != nil {
		return m.MetricLabels
	}
	return nil
}

func (m *WeightedAddr) GetTlsIdentity() *TlsIdentity {
	if m != nil {
		return m.TlsIdentity
	}
	return nil
}

// Which strategy should be used for verifying TLS.
type TlsIdentity struct {
	// Types that are valid to be assigned to Strategy:
	//	*TlsIdentity_K8SPodNamespace_
	Strategy isTlsIdentity_Strategy `protobuf_oneof:"strategy"`
}

func (m *TlsIdentity) Reset()                    { *m = TlsIdentity{} }
func (m *TlsIdentity) String() string            { return proto.CompactTextString(m) }
func (*TlsIdentity) ProtoMessage()               {}
func (*TlsIdentity) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

type isTlsIdentity_Strategy interface{ isTlsIdentity_Strategy() }

type TlsIdentity_K8SPodNamespace_ struct {
	K8SPodNamespace *TlsIdentity_K8SPodNamespace `protobuf:"bytes,1,opt,name=k8s_pod_namespace,json=k8sPodNamespace,oneof"`
}

func (*TlsIdentity_K8SPodNamespace_) isTlsIdentity_Strategy() {}

func (m *TlsIdentity) GetStrategy() isTlsIdentity_Strategy {
	if m != nil {
		return m.Strategy
	}
	return nil
}

func (m *TlsIdentity) GetK8SPodNamespace() *TlsIdentity_K8SPodNamespace {
	if x, ok := m.GetStrategy().(*TlsIdentity_K8SPodNamespace_); ok {
		return x.K8SPodNamespace
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*TlsIdentity) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _TlsIdentity_OneofMarshaler, _TlsIdentity_OneofUnmarshaler, _TlsIdentity_OneofSizer, []interface{}{
		(*TlsIdentity_K8SPodNamespace_)(nil),
	}
}

func _TlsIdentity_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*TlsIdentity)
	// strategy
	switch x := m.Strategy.(type) {
	case *TlsIdentity_K8SPodNamespace_:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.K8SPodNamespace); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("TlsIdentity.Strategy has unexpected type %T", x)
	}
	return nil
}

func _TlsIdentity_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*TlsIdentity)
	switch tag {
	case 1: // strategy.k8s_pod_namespace
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(TlsIdentity_K8SPodNamespace)
		err := b.DecodeMessage(msg)
		m.Strategy = &TlsIdentity_K8SPodNamespace_{msg}
		return true, err
	default:
		return false, nil
	}
}

func _TlsIdentity_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*TlsIdentity)
	// strategy
	switch x := m.Strategy.(type) {
	case *TlsIdentity_K8SPodNamespace_:
		s := proto.Size(x.K8SPodNamespace)
		n += proto.SizeVarint(1<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

// Verify the certificate based on the Kubernetes pod name, and ensure
// that the pod is configured with the same Conduit control plane
// namespace as this proxy.
type TlsIdentity_K8SPodNamespace struct {
	// The Kubernetes namespace of the pod's Conduit control plane.
	ControllerNs string `protobuf:"bytes,1,opt,name=controller_ns,json=controllerNs" json:"controller_ns,omitempty"`
	// The Kubernetes namespace that the pod is in.
	PodNs string `protobuf:"bytes,2,opt,name=pod_ns,json=podNs" json:"pod_ns,omitempty"`
	// The name of the pod.
	PodName string `protobuf:"bytes,3,opt,name=pod_name,json=podName" json:"pod_name,omitempty"`
}

func (m *TlsIdentity_K8SPodNamespace) Reset()                    { *m = TlsIdentity_K8SPodNamespace{} }
func (m *TlsIdentity_K8SPodNamespace) String() string            { return proto.CompactTextString(m) }
func (*TlsIdentity_K8SPodNamespace) ProtoMessage()               {}
func (*TlsIdentity_K8SPodNamespace) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4, 0} }

func (m *TlsIdentity_K8SPodNamespace) GetControllerNs() string {
	if m != nil {
		return m.ControllerNs
	}
	return ""
}

func (m *TlsIdentity_K8SPodNamespace) GetPodNs() string {
	if m != nil {
		return m.PodNs
	}
	return ""
}

func (m *TlsIdentity_K8SPodNamespace) GetPodName() string {
	if m != nil {
		return m.PodName
	}
	return ""
}

type NoEndpoints struct {
	Exists bool `protobuf:"varint,1,opt,name=exists" json:"exists,omitempty"`
}

func (m *NoEndpoints) Reset()                    { *m = NoEndpoints{} }
func (m *NoEndpoints) String() string            { return proto.CompactTextString(m) }
func (*NoEndpoints) ProtoMessage()               {}
func (*NoEndpoints) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func (m *NoEndpoints) GetExists() bool {
	if m != nil {
		return m.Exists
	}
	return false
}

func init() {
	proto.RegisterType((*Update)(nil), "conduit.proxy.destination.Update")
	proto.RegisterType((*AddrSet)(nil), "conduit.proxy.destination.AddrSet")
	proto.RegisterType((*WeightedAddrSet)(nil), "conduit.proxy.destination.WeightedAddrSet")
	proto.RegisterType((*WeightedAddr)(nil), "conduit.proxy.destination.WeightedAddr")
	proto.RegisterType((*TlsIdentity)(nil), "conduit.proxy.destination.TlsIdentity")
	proto.RegisterType((*TlsIdentity_K8SPodNamespace)(nil), "conduit.proxy.destination.TlsIdentity.K8sPodNamespace")
	proto.RegisterType((*NoEndpoints)(nil), "conduit.proxy.destination.NoEndpoints")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for Destination service

type DestinationClient interface {
	// Given a destination, return all addresses in that destination as a long-
	// running stream of updates.
	Get(ctx context.Context, in *conduit_common.Destination, opts ...grpc.CallOption) (Destination_GetClient, error)
}

type destinationClient struct {
	cc *grpc.ClientConn
}

func NewDestinationClient(cc *grpc.ClientConn) DestinationClient {
	return &destinationClient{cc}
}

func (c *destinationClient) Get(ctx context.Context, in *conduit_common.Destination, opts ...grpc.CallOption) (Destination_GetClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_Destination_serviceDesc.Streams[0], c.cc, "/conduit.proxy.destination.Destination/Get", opts...)
	if err != nil {
		return nil, err
	}
	x := &destinationGetClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Destination_GetClient interface {
	Recv() (*Update, error)
	grpc.ClientStream
}

type destinationGetClient struct {
	grpc.ClientStream
}

func (x *destinationGetClient) Recv() (*Update, error) {
	m := new(Update)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Server API for Destination service

type DestinationServer interface {
	// Given a destination, return all addresses in that destination as a long-
	// running stream of updates.
	Get(*conduit_common.Destination, Destination_GetServer) error
}

func RegisterDestinationServer(s *grpc.Server, srv DestinationServer) {
	s.RegisterService(&_Destination_serviceDesc, srv)
}

func _Destination_Get_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(conduit_common.Destination)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(DestinationServer).Get(m, &destinationGetServer{stream})
}

type Destination_GetServer interface {
	Send(*Update) error
	grpc.ServerStream
}

type destinationGetServer struct {
	grpc.ServerStream
}

func (x *destinationGetServer) Send(m *Update) error {
	return x.ServerStream.SendMsg(m)
}

var _Destination_serviceDesc = grpc.ServiceDesc{
	ServiceName: "conduit.proxy.destination.Destination",
	HandlerType: (*DestinationServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Get",
			Handler:       _Destination_Get_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "proxy/destination/destination.proto",
}

func init() { proto.RegisterFile("proxy/destination/destination.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 548 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xa4, 0x54, 0xdd, 0x6e, 0xd3, 0x30,
	0x18, 0x6d, 0xda, 0x2d, 0x6b, 0xbf, 0xb4, 0x2a, 0x33, 0x3f, 0xca, 0xca, 0xcd, 0xc8, 0x04, 0x4c,
	0x5c, 0x64, 0x53, 0x91, 0x50, 0x81, 0x01, 0x62, 0x62, 0xa2, 0xd5, 0xa0, 0x42, 0x66, 0x08, 0xae,
	0x88, 0xbc, 0xd8, 0xda, 0xa2, 0x26, 0x76, 0x64, 0xbb, 0x63, 0x7d, 0x3d, 0xde, 0x83, 0x07, 0xe0,
	0x9e, 0x07, 0x40, 0x75, 0x5c, 0x9a, 0x15, 0xd1, 0x55, 0xe2, 0x2a, 0xfe, 0xac, 0x73, 0x8e, 0xbf,
	0x73, 0x3e, 0xc7, 0xb0, 0x93, 0x4b, 0x71, 0x39, 0xd9, 0xa3, 0x4c, 0xe9, 0x84, 0x13, 0x9d, 0x08,
	0x5e, 0x5e, 0x87, 0xb9, 0x14, 0x5a, 0xa0, 0xad, 0x58, 0x70, 0x3a, 0x4e, 0x74, 0x68, 0xc0, 0x61,
	0x09, 0xd0, 0xb9, 0x19, 0x8b, 0x2c, 0x13, 0x7c, 0xaf, 0xf8, 0x14, 0xf8, 0xe0, 0x87, 0x03, 0xee,
	0xa7, 0x9c, 0x12, 0xcd, 0xd0, 0x4b, 0xa8, 0x11, 0x4a, 0x7d, 0x67, 0xdb, 0xd9, 0xf5, 0xba, 0x8f,
	0xc2, 0x7f, 0x0a, 0x85, 0x9f, 0x59, 0x72, 0x76, 0xae, 0x19, 0x7d, 0x4d, 0xa9, 0xfc, 0xc8, 0x74,
	0xbf, 0x82, 0xa7, 0x44, 0x74, 0x00, 0xae, 0x64, 0x99, 0xb8, 0x60, 0x7e, 0xd5, 0x48, 0x04, 0x4b,
	0x24, 0xe6, 0x54, 0xcb, 0x41, 0xc7, 0xd0, 0xe4, 0x22, 0x62, 0x9c, 0xe6, 0x22, 0xe1, 0x5a, 0xf9,
	0x35, 0xa3, 0xf1, 0x60, 0x89, 0xc6, 0x50, 0x1c, 0xcd, 0xd0, 0xfd, 0x0a, 0xf6, 0xf8, 0xbc, 0x3c,
	0xac, 0x83, 0x3b, 0x36, 0xa6, 0x82, 0xe7, 0xb0, 0x61, 0xcf, 0x42, 0xfb, 0xb0, 0x4e, 0x28, 0x95,
	0xca, 0x77, 0xb6, 0x6b, 0xbb, 0x5e, 0xb7, 0xf3, 0x47, 0xda, 0x06, 0x72, 0x12, 0xe7, 0x53, 0x28,
	0x53, 0x0a, 0x17, 0xc0, 0xe0, 0x97, 0x03, 0xed, 0x05, 0xb3, 0xe8, 0xc5, 0x55, 0x95, 0x87, 0x2b,
	0xe6, 0x64, 0x25, 0x11, 0x81, 0x56, 0xc6, 0xb4, 0x4c, 0xe2, 0x28, 0x25, 0xa7, 0x2c, 0x55, 0x7e,
	0xd5, 0xc8, 0x1c, 0xac, 0x1e, 0x77, 0xf8, 0xde, 0xf0, 0xdf, 0x19, 0xfa, 0x11, 0xd7, 0x72, 0x82,
	0x9b, 0x59, 0x69, 0xab, 0xf3, 0x0a, 0x36, 0xff, 0x82, 0xa0, 0x1b, 0x50, 0x1b, 0xb1, 0x89, 0x19,
	0x6e, 0x03, 0x4f, 0x97, 0xe8, 0x16, 0xac, 0x5f, 0x90, 0x74, 0x5c, 0x4c, 0xab, 0x81, 0x8b, 0xe2,
	0x59, 0xb5, 0xe7, 0x04, 0xdf, 0xab, 0xd0, 0x2c, 0x1f, 0x8a, 0x42, 0x58, 0x9b, 0x76, 0x6f, 0xaf,
	0xc6, 0xb2, 0xe0, 0x0c, 0x0e, 0xdd, 0x01, 0xf7, 0x9b, 0xe1, 0x9b, 0x29, 0xb6, 0xb0, 0xad, 0xd0,
	0xd7, 0x45, 0xf3, 0x6b, 0xc6, 0xfc, 0xd3, 0x15, 0xcd, 0x5f, 0xe7, 0x1c, 0x0d, 0xa0, 0xa9, 0x53,
	0x15, 0x25, 0x94, 0x71, 0x9d, 0xe8, 0x89, 0xbf, 0x7e, 0xed, 0x1d, 0x3a, 0x49, 0xd5, 0xc0, 0xa2,
	0xb1, 0xa7, 0xe7, 0xc5, 0xff, 0x87, 0xf8, 0xd3, 0x01, 0xaf, 0xa4, 0x8e, 0x28, 0x6c, 0x8e, 0x7a,
	0x2a, 0xca, 0x05, 0x8d, 0x38, 0xc9, 0x98, 0xca, 0x49, 0xcc, 0x6c, 0xa0, 0x4f, 0x56, 0x6b, 0x30,
	0x3c, 0xee, 0xa9, 0x0f, 0x82, 0x0e, 0x67, 0xec, 0x7e, 0x05, 0xb7, 0x47, 0x57, 0xb7, 0x3a, 0xe7,
	0xd0, 0x5e, 0x40, 0xa1, 0x1d, 0x68, 0xc5, 0x82, 0x6b, 0x29, 0xd2, 0x94, 0xc9, 0x88, 0x2b, 0xdb,
	0x7e, 0x73, 0xbe, 0x39, 0x54, 0xe8, 0x36, 0xb8, 0xa6, 0x33, 0x35, 0x33, 0x92, 0x0b, 0x3a, 0x54,
	0x68, 0x0b, 0xea, 0xb3, 0x86, 0xcd, 0x28, 0x1b, 0x78, 0x23, 0x2f, 0xb4, 0x0f, 0x01, 0xea, 0x4a,
	0x4b, 0xa2, 0xd9, 0xd9, 0x24, 0xb8, 0x0f, 0x5e, 0xe9, 0x67, 0x9c, 0x8e, 0x9f, 0x5d, 0x26, 0x4a,
	0x17, 0x47, 0xd5, 0xb1, 0xad, 0xba, 0x5f, 0xc0, 0x7b, 0x33, 0xb7, 0x86, 0x06, 0x50, 0x7b, 0xcb,
	0x34, 0xba, 0xbb, 0x78, 0x9d, 0x4a, 0x98, 0xce, 0xbd, 0x25, 0xd1, 0x14, 0xcf, 0x56, 0x50, 0xd9,
	0x77, 0x4e, 0x5d, 0xf3, 0x98, 0x3d, 0xfe, 0x1d, 0x00, 0x00, 0xff, 0xff, 0x60, 0xc3, 0x5a, 0x03,
	0x23, 0x05, 0x00, 0x00,
}
