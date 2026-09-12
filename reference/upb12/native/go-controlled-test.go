package controlled

import("bytes";"log";"math";"testing")

func TestUPB12NativeCorpus(t *testing.T){oldWriter,oldFlags:=log.Writer(),log.Flags();defer func(){log.SetOutput(oldWriter);log.SetFlags(oldFlags)}();var trace bytes.Buffer;log.SetOutput(&trace);log.SetFlags(0);for i:=int64(0);i<4096;i++{state,delta:=i*7919-10000000,i*104729-200000000;if i%257==0{state,delta=math.MaxInt64,1};trace.Reset();got:=DispatchControlled(ControlledState{Value:state},ControlledCommand{Delta:delta});if got.Value!=state+delta||trace.String()!="true\n"{t.Fatalf("observation %d: value=%d trace=%q",i,got.Value,trace.String())}}}
