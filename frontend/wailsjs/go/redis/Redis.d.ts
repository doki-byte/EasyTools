// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT
import {define} from '../models';
import {redis} from '../models';
import {context} from '../models';

export function ConnectionCreate(arg1:define.Connection):Promise<redis.H>;

export function ConnectionDelete(arg1:string):Promise<redis.H>;

export function ConnectionEdit(arg1:define.Connection):Promise<redis.H>;

export function ConnectionList():Promise<redis.H>;

export function CreateKeyValue(arg1:define.CreateKeyValueRequest):Promise<redis.H>;

export function DbInfo(arg1:string):Promise<redis.H>;

export function DbList(arg1:string):Promise<redis.H>;

export function DeleteKeyValue(arg1:define.KeyValueRequest):Promise<redis.H>;

export function GetKeyValue(arg1:define.KeyValueRequest):Promise<redis.H>;

export function HashAddOrUpdateField(arg1:define.HashAddOrUpdateFieldRequest):Promise<redis.H>;

export function HashFieldDelete(arg1:define.HashFieldDeleteRequest):Promise<redis.H>;

export function KeyList(arg1:define.KeyListRequest):Promise<redis.H>;

export function ListValueCreate(arg1:define.ListValueRequest):Promise<redis.H>;

export function ListValueDelete(arg1:define.ListValueRequest):Promise<redis.H>;

export function SetCtx(arg1:context.Context):Promise<void>;

export function SetValueCreate(arg1:define.SetValueRequest):Promise<redis.H>;

export function SetValueDelete(arg1:define.SetValueRequest):Promise<redis.H>;

export function UpdateKeyValue(arg1:define.UpdateKeyValueRequest):Promise<redis.H>;

export function ZSetValueCreate(arg1:define.ZSetValueRequest):Promise<redis.H>;

export function ZSetValueDelete(arg1:define.ZSetValueRequest):Promise<redis.H>;
