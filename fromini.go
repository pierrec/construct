package iniconfig

import ini "github.com/pierrec/go-ini"

func newIni() (*ini.INI, error) {
	return ini.New(ini.Comment("# "))
}

func iniLoad(from FromIni) (*ini.INI, error) {
	if from == nil {
		return nil, nil
	}
	src, err := from.LoadConfig()
	if err != nil {
		return nil, err
	}
	if src == nil {
		return nil, nil
	}
	defer src.Close()

	inic, err := newIni()
	if _, err := inic.ReadFrom(src); err != nil {
		return nil, err
	}
	return inic, nil
}

func iniSave(inic *ini.INI, ini FromIni) error {
	dest, err := ini.WriteConfig()
	if err != nil || dest == nil {
		return err
	}
	_, err = inic.WriteTo(dest)
	if err != nil {
		return err
	}
	return dest.Close()
}

func (c *config) iniAddComments(ini *ini.INI) {
	ini.SetComments("", "", c.raw.UsageConfig(""))

	for _, section := range append(ini.Sections(), "") {
		if section != "" {
			usage := c.usage(section)
			ini.SetComments(section, "", usage)
		}

		for _, key := range ini.Keys(section) {
			name := toName(section, key)
			usage := c.usage(name)
			ini.SetComments(section, key, usage)
		}
	}
}

func (c *config) iniRemoveHidden(ini *ini.INI) {
	for _, section := range append(ini.Sections(), "") {
		for _, key := range ini.Keys(section) {
			name := toName(section, key)
			usage := c.usage(name)
			if usage == "" {
				ini.Del(section, key)
			}
		}
	}
}
