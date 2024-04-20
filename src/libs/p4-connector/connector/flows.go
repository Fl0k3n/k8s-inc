package connector

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Fl0k3n/k8s-inc/libs/p4r/client"
	"github.com/Fl0k3n/k8s-inc/libs/p4r/util/conversion"

	p4_config_v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
)


type MatchEntry = map[string]client.MatchInterface
type MatchFieldName = string
type ActionParamName = string

type ActionParam = []byte

var (
	IP_V4_REGEX = regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
	MAC_REGEX   = regexp.MustCompile(`([\da-fA-F]{2}:){5}[\da-fA-F]{2}`)
)

const (
	LPM_SEPARATOR = "/"
	TERNARY_SEPARATOR = "&&&"
)

type ActionEntry struct {
	Name string
	Params []ActionParam
}

type TableEntry struct {
	TableName string
	Match MatchEntry
	Action ActionEntry
}

type RawTableEntry struct {
	TableName string
	Match map[string]string
	ActionName string
	ActionParams map[string]string
}

type MetadataNotFoundError struct {
	What string
	Where string
}

func (m *MetadataNotFoundError) Error() string {
	return fmt.Sprintf("%s not found in %s", m.What, m.Where)
}

type InvalidArgumentError struct {
	Msg string
}

func (i *InvalidArgumentError) Error() string {
	return i.Msg
}

func newNotFoundError(what string, where string) error {
	return &MetadataNotFoundError{What: what, Where: where}
}

func newInvalidArgumentError(msg string) error {
	return &InvalidArgumentError{Msg: msg}
}

func (p *P4RuntimeConnector) action(actionName string) (*p4_config_v1.Action, error) {
	for _, act := range p.p4info.Actions {
		if act.Preamble.Name == actionName || act.Preamble.Alias == actionName {
			return act, nil
		}
	}
	return nil, newNotFoundError("action: " + actionName, "p4info actions")
}

func (p *P4RuntimeConnector) table(tableName string) (*p4_config_v1.Table, error) {
	for _, tab := range p.p4info.Tables {
		if tab.Preamble.Name == tableName || tab.Preamble.Alias == tableName{
			return tab, nil
		}
	}
	return nil, newNotFoundError("table: " + tableName, "p4info tables")
}

func getActionParamsMap(action *p4_config_v1.Action) map[ActionParamName]*p4_config_v1.Action_Param {
	res := map[ActionParamName]*p4_config_v1.Action_Param{}
	for _, param := range action.Params {
		res[param.Name] = param
	}
	return res
}

func getMatchMap(table *p4_config_v1.Table) map[MatchFieldName]*p4_config_v1.MatchField {
	res := map[MatchFieldName]*p4_config_v1.MatchField{}
	for _, mf := range table.MatchFields {
		res[mf.Name] = mf
	}
	return res
}

func actionParamOrMatchValueToBytes(val string, bitwidth int) ([]byte, error) {
	if bitwidth == 32 && IP_V4_REGEX.MatchString(val) {
		return conversion.IpToBinary(val)
	} else if bitwidth == 48 && MAC_REGEX.MatchString(val) {
		return conversion.MacToBinary(val)	
	} else {
		return conversion.StringToBinaryUint(val, bitwidth)
	}
}

func ternaryMaskToBytes(val string, bitwidth int) ([]byte, error) {
	u, e := strconv.ParseUint(val, 0, bitwidth)
	if e != nil {
		return nil, e
	}	
	return conversion.NumberToBinaryUint(u, bitwidth), nil
}

func (p *P4RuntimeConnector) BuildActionEntry(actionName string, rawParams map[ActionParamName]string) (ActionEntry, error) {
	if p.p4info == nil {
		panic(errors.New("P4Info unset"))
	}
	action, err := p.action(actionName)
	if err != nil {
		return ActionEntry{}, err
	}
	actionParams := getActionParamsMap(action)
	if len(actionParams) != len(rawParams) {
		return ActionEntry{}, newInvalidArgumentError(fmt.Sprintf(
			"action: %s\tInvalid number of parameters, expected %d, got %d",
			actionName, len(actionParams), len(rawParams)),
		)
	}

	res := make([]ActionParam, len(rawParams))

	for paramName, val := range rawParams {
		if param, ok := actionParams[paramName]; ok {
			bytes, err := actionParamOrMatchValueToBytes(val, int(param.Bitwidth))
			if err != nil {
				err = newInvalidArgumentError(fmt.Sprintf(
					"action: %s\tFailed to parse value for param %s, error: %e", actionName, paramName, err))
				return ActionEntry{}, err
			}
			res[param.Id - 1] = bytes
		} else {
			return ActionEntry{}, newNotFoundError("param: " + paramName, "params of action: " + actionName)
		}
	}
	return ActionEntry{Name: actionName, Params: res}, nil
}

func (p *P4RuntimeConnector) BuildMatchEntry(tableName string, rawMatchEntries map[MatchFieldName]string) (MatchEntry, error) {
	table, err := p.table(tableName)
	if err != nil {
		return nil, err
	}
	res := make(MatchEntry)
	mfs := getMatchMap(table)

	for key, matchValue := range rawMatchEntries {
		if mf, ok := mfs[key]; ok {
			var val []byte
			var matchVal client.MatchInterface
			var err error
			switch mf.Match.(*p4_config_v1.MatchField_MatchType_).MatchType {
			case p4_config_v1.MatchField_EXACT: // 1.1.1.1
				val, err = actionParamOrMatchValueToBytes(matchValue, int(mf.Bitwidth))
				matchVal = &client.ExactMatch{
					Value: val,
				}
			case p4_config_v1.MatchField_LPM:   // 1.1.1.1/24
				parts := strings.Split(matchValue, LPM_SEPARATOR)
				if len(parts) != 2 {
					return nil, newInvalidArgumentError(fmt.Sprintf(
						"Invalid LPM format, should be value%sprefix_size, got %s", LPM_SEPARATOR, matchValue))
				}
				val, err = actionParamOrMatchValueToBytes(parts[0], int(mf.Bitwidth))
				if err == nil {
					var lpm uint64
					lpm, err = strconv.ParseUint(parts[1], 10, 6) // at most 64 bits, 2^6 is enough to store it
					matchVal = &client.LpmMatch{
						Value: val,
						PLen: int32(lpm),
					}
				}
			case p4_config_v1.MatchField_TERNARY: //1.1.1.1&&&0xFFF00FFF, simple_switch_CLI syntax
				parts := strings.Split(matchValue, TERNARY_SEPARATOR)
				if len(parts) != 2 {
					return nil, newInvalidArgumentError(fmt.Sprintf(
						"Invalid ternary format, should be value%smask, got %s", TERNARY_SEPARATOR, matchValue))
				}
				val, err = actionParamOrMatchValueToBytes(parts[0], int(mf.Bitwidth))
				if err == nil {
					var mask []byte
					mask, err = ternaryMaskToBytes(parts[1], int(mf.Bitwidth))
					matchVal = &client.TernaryMatch{
						Value: val,
						Mask: mask,
					}
				}
			default:
				return nil, newInvalidArgumentError("Unsupported match type")
			}
			if err != nil {
				return nil, newInvalidArgumentError(fmt.Sprintf(
					"table: %s\tFailed to parse key: %s, got: %s", tableName, key, matchValue))
			}
			res[key] = matchVal
		} else {
			return res, newNotFoundError("match key: " + key, "keys of table " + tableName)
		}
	}

	return res, nil
}


func (p *P4RuntimeConnector) mapToProtobufTableEntry(entry TableEntry) *p4_v1.TableEntry {
	defaultPriority := int32(0)
	for _, matchValue := range entry.Match {
		if _, isTernary := matchValue.(*client.TernaryMatch); isTernary {
			defaultPriority = 2147483646 // TODO this is 1 on switch, for some reason it is negated (unsigned 31 bits are negated)
			break
		}
	}
	return p.p4rtc.NewTableEntry(
		entry.TableName,
		entry.Match,
		p.p4rtc.NewTableActionDirect(entry.Action.Name, entry.Action.Params),
		&client.TableEntryOptions{
			Priority: defaultPriority,
			IdleTimeout: 0, // no timeout
		},
	)
}


func (p *P4RuntimeConnector) WriteMatchActionEntry(ctx context.Context, entry TableEntry) error {
	return p.p4rtc.InsertTableEntry(ctx, p.mapToProtobufTableEntry(entry))
}

func (p *P4RuntimeConnector) DeleteMatchActionEntry(ctx context.Context, entry TableEntry) error {
	return p.p4rtc.DeleteTableEntry(ctx, p.mapToProtobufTableEntry(entry))
}
