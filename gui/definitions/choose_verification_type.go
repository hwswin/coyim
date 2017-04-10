package definitions

func init() {
	add(`ChooseVerificationType`, &defChooseVerificationType{})
}

type defChooseVerificationType struct{}

func (*defChooseVerificationType) String() string {
	return `<interface>
  <object class="GtkDialog" id="dialog">
    <property name="window-position">GTK_WIN_POS_CENTER</property>
    <child internal-child="vbox">
      <object class="GtkBox" id="box">
        <property name="border-width">10</property>
        <property name="homogeneous">false</property>
        <property name="orientation">GTK_ORIENTATION_VERTICAL</property>
        <child>
          <object  class="GtkImage">
            <property name="file">build/images/maninthemiddle.png</property>
          </object>
          <packing>
            <property name="padding">20</property>
          </packing>
        </child>
        <child>
          <object class="GtkLabel">
            <property name="label" translatable="yes">Make sure there is no one else reading your messages.</property>
          </object>
        </child>
        <child>
          <object class="GtkRadioButton" id="verification_type">
            <property name="label" translatable="yes">Use a shared PIN</property>
            <signal name="touch-event" handler="use_smp"/>
          </object>
        </child>
        <child>
          <object class="GtkRadioButton">
            <property name="label" translatable="yes">Verify their fingerprint</property>
            <property name="group">verification_type</property>
            <signal name="touch-event" handler="use_fingerprint"/>
          </object>
        </child>
        <child internal-child="action_area">
          <object class="GtkButtonBox" id="button_box">
            <property name="orientation">GTK_ORIENTATION_HORIZONTAL</property>
            <child>
              <object class="GtkButton" id="cancel_button">
                <property name="label" translatable="yes">Cancel</property>
              </object>
            </child>
            <child>
              <object class="GtkButton" id="validate_button">
                <property name="label" translatable="yes">Validate</property>
              </object>
            </child>
          </object>
        </child>
      </object>
    </child>
  </object>
</interface>
`
}