import assert from "node:assert/strict";
import {ControlledCommand,ControlledState,DispatchControlled} from "./controlled.js";
for(let i=0n;i<4096n;i++){let state=i*7919n-10000000n,delta=i*104729n-200000000n;if(i%257n===0n){state=9223372036854775807n;delta=1n;}const trace=[],original=console.log;console.log=value=>trace.push(value);let result;try{result=DispatchControlled(new ControlledState(state),new ControlledCommand(delta));}finally{console.log=original;}assert.equal(result.Value,BigInt.asIntN(64,state+delta),`observation ${i}`);assert.deepEqual(trace,[true]);}
console.log("JavaScript UPB12 native project passes 4,096 observations");
